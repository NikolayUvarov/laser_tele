package proxy

// Media connections through Cloudflare domains over HTTP/2: every MTProto packet of the client
// is sent as POST https://kwsN.<domain>/api (the HTTP transport of MTProto), many clients share
// one HTTP/2 connection per domain. Because the HTTP transport answers only requests, the last
// encrypted packet is sent again as a poll when the channel is idle; Telegram ignores the
// duplicate message and answers with pending data. Ported from cf_h2.py of tg-ws-proxy

import (
	"bytes"
	"context"
	"crypto/cipher"
	"crypto/rand"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/logx"
)

const (
	h2MaxPacket           = 4 * 1024 * 1024
	h2MaxPlainPadding     = 256
	h2MaxChannelRequests  = 8
	h2MaxChannelBytes     = 8 * 1024 * 1024
	h2MaxLaneRequests     = 64
	h2MaxLaneBytes        = 32 * 1024 * 1024
	h2MaxLaneReplyBytes   = 32 * 1024 * 1024
	h2DomainCooldown      = 30 * time.Second
	h2RequestTimeout      = 65 * time.Second
	h2SetupTimeout        = 8 * time.Second
	replayIdle            = time.Second
	replayRequest         = 3 * time.Second
	replayCheck           = 250 * time.Millisecond
	replayMaxAge          = 30 * time.Second
	replayHistoryBytes    = 64 * 1024
	replayHistoryPackets  = 16
	maxChannelReplays     = 2
	replaySlot            = 30 * time.Second
	replayRetry           = 2 * time.Second
	replayMaxAttempts     = 3
	h2ReplyQueueSize      = 16
	h2SlowResponseSeconds = 3 * time.Second
)

var (
	errH2ChannelClosed = errors.New("H2 channel closed")
	errH2LaneClosed    = errors.New("H2 lane closed")
	errReplyBufferFull = errors.New("H2 lane response buffers exceeded bound")
	errH2Canceled      = errors.New("H2 request canceled")
)

// mtprotoTransportError is an error code the native client must receive, not just EOF
type mtprotoTransportError struct {
	code   int32
	source string
}

func (e *mtprotoTransportError) Error() string { return fmt.Sprintf("%d (%s)", e.code, e.source) }

// readNativePacket reads one MTProto transport packet of the client and returns its body
func readNativePacket(r io.Reader, dec cipher.Stream, tag [4]byte) (body []byte, quick bool, err error) {
	plaintext := func(n int) ([]byte, error) {
		buf := make([]byte, n)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		dec.XORKeyStream(buf, buf)
		return buf, nil
	}
	var length int
	if tag == tagAbridged {
		first, err := plaintext(1)
		if err != nil {
			return nil, false, err
		}
		quick = first[0]&0x80 != 0
		words := int(first[0] & 0x7f)
		if words == 0x7f {
			b, err := plaintext(3)
			if err != nil {
				return nil, false, err
			}
			words = int(b[0]) | int(b[1])<<8 | int(b[2])<<16
		}
		length = words * 4
	} else {
		b, err := plaintext(4)
		if err != nil {
			return nil, false, err
		}
		value := binary.LittleEndian.Uint32(b)
		quick = value&0x80000000 != 0
		length = int(value & 0x7fffffff)
	}
	limit := h2MaxPacket
	if tag == tagPadded {
		limit += h2MaxPlainPadding
	}
	if length < 24 || length > limit {
		return nil, false, fmt.Errorf("native packet length outside supported range: %d", length)
	}
	body, err = plaintext(length)
	if err != nil {
		return nil, false, err
	}
	if tag == tagPadded {
		var packetLen, maxPadding int
		if bytes.Equal(body[:8], make([]byte, 8)) {
			packetLen = 20 + int(binary.LittleEndian.Uint32(body[16:20]))
			maxPadding = h2MaxPlainPadding
		} else {
			packetLen = 24 + ((length-24)/16)*16
			maxPadding = 15
		}
		padding := length - packetLen
		if packetLen < 24 || packetLen > length || padding > maxPadding {
			return nil, false, fmt.Errorf("invalid padded native packet: frame=%d packet=%d padding=%d limit=%d",
				length, packetLen, padding, maxPadding)
		}
		body = body[:packetLen]
	}
	if len(body)%4 != 0 || len(body) > h2MaxPacket {
		return nil, false, errors.New("invalid native packet alignment or size")
	}
	return body, quick, nil
}

// encodeReply frames a response body for the native client
func encodeReply(body []byte, tag [4]byte) ([]byte, error) {
	if len(body) == 0 || len(body)%4 != 0 || len(body) > h2MaxPacket {
		return nil, fmt.Errorf("invalid HTTP MTProto response length: %d", len(body))
	}
	if tag == tagAbridged {
		words := len(body) / 4
		var out []byte
		if words < 0x7f {
			out = append(make([]byte, 0, 1+len(body)), byte(words))
		} else {
			out = append(make([]byte, 0, 4+len(body)), 0x7f, byte(words), byte(words>>8), byte(words>>16))
		}
		return append(out, body...), nil
	}
	var padding []byte
	if tag == tagPadded {
		var n [1]byte
		rand.Read(n[:])
		padding = make([]byte, int(n[0])%4)
		rand.Read(padding)
	}
	out := binary.LittleEndian.AppendUint32(make([]byte, 0, 4+len(body)+len(padding)), uint32(len(body)+len(padding)))
	out = append(out, body...)
	return append(out, padding...), nil
}

// waitCond waits under mu until cond is true; mu is held on return
func waitCond(ctx context.Context, mu *sync.Mutex, notify *chan struct{}, cond func() bool, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for !cond() {
		ch := *notify
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return context.DeadlineExceeded
		}
		mu.Unlock()
		timer := time.NewTimer(remaining)
		select {
		case <-ch:
		case <-timer.C:
		case <-ctx.Done():
		}
		timer.Stop()
		mu.Lock()
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	return nil
}

func bump(notify *chan struct{}) {
	close(*notify)
	*notify = make(chan struct{})
}

type replayPacket struct {
	sentAt        time.Time
	body          []byte
	sequence      int
	attempts      int
	lastAttemptAt time.Time
	pending       *h2Req
	originals     map[*h2Req]struct{}
	retired       bool
	replyID       []byte
}

func (p *replayPacket) replayed() bool { return !p.lastAttemptAt.IsZero() || p.attempts > 0 }

type h2Req struct {
	cancel    context.CancelFunc
	done      chan struct{}
	length    int
	since     time.Time
	sentAt    time.Time // zero until the body was sent
	receiving bool      // response headers arrived, body is being read
	replay    bool
	packet    *replayPacket
}

type h2Item struct {
	data []byte
	err  error
}

type h2Channel struct {
	lane  *h2Lane
	id    int
	label string
	ctx   context.Context
	stop  context.CancelFunc

	sendLock sync.Mutex

	mu               sync.Mutex
	notify           chan struct{}
	closed           bool
	transportErr     *mtprotoTransportError
	pending          map[*h2Req]struct{}
	pendingBytes     int
	replyBytes       int
	queue            chan h2Item
	wakeup           chan struct{}
	deliveringSince  time.Time
	history          []*replayPacket
	replayKey        []byte
	replayPending    map[*h2Req]struct{}
	lastProgress     time.Time
	lastNativeSend   time.Time
	uploadWaitSince  time.Time
	uploadWaitPhase  string
	uploadWaitBytes  int
	requests         int
	replayRequests   int
	replayRotations  int
	capacityRotation int
	up, down         int
	started          time.Time
	closeOnce        sync.Once
}

func newH2Channel(lane *h2Lane, id int, label string) *h2Channel {
	now := time.Now()
	ctx, stop := context.WithCancel(context.Background())
	c := &h2Channel{lane: lane, id: id, label: label, ctx: ctx, stop: stop, notify: make(chan struct{}),
		pending: map[*h2Req]struct{}{}, queue: make(chan h2Item, h2ReplyQueueSize), wakeup: make(chan struct{}, 1),
		replayPending: map[*h2Req]struct{}{}, lastProgress: now, lastNativeSend: now, started: now, uploadWaitPhase: "-"}
	lane.addChannel(c)
	return c
}

func (c *h2Channel) nativeBackpressure() bool { return c.replyBytes > 0 || !c.deliveringSince.IsZero() }

func (c *h2Channel) hasCapacity(n int) bool {
	return len(c.pending) < h2MaxChannelRequests && c.pendingBytes+n <= h2MaxChannelBytes
}

func (c *h2Channel) wakeReceiverLocked() {
	if !c.closed && len(c.pending) == 0 && !c.nativeBackpressure() {
		select {
		case c.wakeup <- struct{}{}:
		default:
		}
	}
}

func (c *h2Channel) lastEntry() *replayPacket {
	if len(c.history) == 0 {
		return nil
	}
	return c.history[len(c.history)-1]
}

func (c *h2Channel) overdueOriginal(entry *replayPacket, now time.Time) bool {
	if entry.retired {
		return true
	}
	for req := range entry.originals {
		if !req.sentAt.IsZero() && !req.receiving && now.Sub(req.sentAt) >= replayRequest {
			return true
		}
	}
	return false
}

func (c *h2Channel) replayLimit(entry *replayPacket) int {
	if len(entry.originals) > 0 || entry.retired || entry == c.lastEntry() {
		return replayMaxAttempts
	}
	return 0
}

// replayBlocker returns why the entry can't be replayed now, "" if it can
func (c *h2Channel) replayBlocker(entry *replayPacket, now time.Time) string {
	if age := now.Sub(entry.sentAt); age < 0 || age > replayMaxAge {
		return "expired"
	}
	if entry.attempts >= c.replayLimit(entry) {
		return "budget-used"
	}
	if c.nativeBackpressure() {
		return "native-backpressure"
	}
	for req := range entry.originals {
		if req.receiving {
			return "original-body"
		}
	}
	if entry.pending != nil && entry.pending.receiving {
		return "replay-body"
	}
	if entry.replayed() && now.Sub(entry.lastAttemptAt) < replayRetry {
		return "backoff-or-progress"
	}
	if c.overdueOriginal(entry, now) {
		return ""
	}
	if entry != c.lastEntry() {
		return "receiver-active"
	}
	for req := range c.pending {
		if req != entry.pending {
			return "receiver-active"
		}
	}
	if entry.replayed() && now.Sub(c.lastProgress) < replayIdle {
		return "backoff-or-progress"
	}
	return ""
}

func (c *h2Channel) replayDue(entry *replayPacket, now time.Time) bool {
	return c.replayBlocker(entry, now) == ""
}

func (c *h2Channel) sortedPending() []*h2Req {
	reqs := make([]*h2Req, 0, len(c.pending))
	for req := range c.pending {
		reqs = append(reqs, req)
	}
	sort.Slice(reqs, func(i, j int) bool { return reqs[i].since.Before(reqs[j].since) })
	return reqs
}

// capacityCandidate finds a request whose answer is overdue and that can be cancelled to make room
func (c *h2Channel) capacityCandidate(now time.Time) (*h2Req, *replayPacket) {
	var sent []*h2Req
	for req := range c.pending {
		if !req.sentAt.IsZero() {
			sent = append(sent, req)
		}
	}
	sort.Slice(sent, func(i, j int) bool {
		if sent[i].replay != sent[j].replay {
			return sent[i].replay
		}
		return sent[i].sentAt.Before(sent[j].sentAt)
	})
	for _, req := range sent {
		if req.receiving || now.Sub(req.sentAt) < replayRequest {
			continue
		}
		if req.replay {
			return req, nil
		}
		if entry := req.packet; entry != nil {
			if _, ok := entry.originals[req]; ok && now.Sub(entry.sentAt) <= replayMaxAge {
				return req, entry
			}
		}
	}
	return nil, nil
}

func (c *h2Channel) oldestSent(now time.Time) time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	oldest := now
	for req := range c.pending {
		if !req.sentAt.IsZero() && req.sentAt.Before(oldest) {
			oldest = req.sentAt
		}
	}
	return oldest
}

// makeUploadRoom cancels an overdue request when a new packet of the client waits for capacity
func (c *h2Channel) makeUploadRoom(now time.Time) {
	c.mu.Lock()
	if c.closed || c.uploadWaitSince.IsZero() {
		c.mu.Unlock()
		return
	}
	phase, waitBytes := c.uploadWaitPhase, c.uploadWaitBytes
	var channels []*h2Channel
	switch phase {
	case "channel-capacity":
		if c.hasCapacity(waitBytes) {
			c.mu.Unlock()
			return
		}
		c.mu.Unlock()
		channels = []*h2Channel{c}
	case "lane-capacity":
		c.mu.Unlock()
		if c.lane.hasCapacity(waitBytes) {
			return
		}
		channels = c.lane.channelList()
		oldest := map[*h2Channel]time.Time{}
		for _, ch := range channels {
			oldest[ch] = ch.oldestSent(now)
		}
		sort.Slice(channels, func(i, j int) bool { return oldest[channels[i]].Before(oldest[channels[j]]) })
	default:
		c.mu.Unlock()
		return
	}
	for _, ch := range channels {
		ch.mu.Lock()
		req, entry := ch.capacityCandidate(now)
		if req == nil {
			ch.mu.Unlock()
			continue
		}
		req.sentAt = time.Time{}
		if entry != nil {
			entry.retired = true
		}
		ch.capacityRotation++
		ch.mu.Unlock()
		logx.Debugf("H2 CAPACITY lane=%d waiting_channel=%d release_channel=%d reason=%s replay=%v",
			c.lane.id, c.id, ch.id, phase, req.replay)
		req.cancel()
		<-req.done
		c.lane.capacityRotations.Add(1)
		return
	}
}

// send starts a request with body; replays never wait and return false when not possible
func (c *h2Channel) send(body []byte, quick, replay bool) (bool, error) {
	if replay {
		if !c.sendLock.TryLock() {
			return false, nil
		}
	} else {
		c.mu.Lock()
		c.uploadWaitSince = time.Now()
		c.uploadWaitPhase = "send-lock"
		c.uploadWaitBytes = len(body)
		c.mu.Unlock()
		c.sendLock.Lock()
		defer func() {
			c.mu.Lock()
			c.uploadWaitSince = time.Time{}
			c.uploadWaitPhase = "-"
			c.uploadWaitBytes = 0
			c.mu.Unlock()
		}()
	}
	defer c.sendLock.Unlock()
	return c.doSend(body, replay)
}

func (c *h2Channel) doSend(body []byte, replay bool) (bool, error) {
	n := len(body)
	c.mu.Lock()
	if replay && !c.hasCapacity(n) {
		c.mu.Unlock()
		return false, nil
	}
	if !replay {
		c.uploadWaitPhase = "channel-capacity"
		if err := waitCond(c.ctx, &c.mu, &c.notify, func() bool { return c.closed || c.hasCapacity(n) }, h2RequestTimeout); err != nil {
			c.mu.Unlock()
			if c.ctx.Err() != nil {
				return false, errH2ChannelClosed
			}
			return false, errors.New("H2 channel capacity wait timed out")
		}
	}
	if c.closed {
		c.mu.Unlock()
		return false, errH2ChannelClosed
	}
	if !replay {
		c.uploadWaitPhase = "lane-capacity"
	}
	c.mu.Unlock()

	ok, err := c.lane.reserve(c.ctx, n, !replay)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}

	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		c.lane.release(n)
		return false, errH2ChannelClosed
	}
	now := time.Now()
	var packet *replayPacket
	if replay {
		packet = c.findPacket(body)
		if packet == nil || packet.pending != nil || !c.replayDue(packet, now) {
			c.mu.Unlock()
			c.lane.release(n)
			return false, nil
		}
	}
	c.pendingBytes += n
	c.requests++
	if replay {
		c.replayRequests++
	}
	c.up += n
	var native *replayPacket
	if !replay {
		c.lastNativeSend = now
		native = c.rememberPacket(body, now)
		packet = native
	}
	reqCtx, cancel := context.WithCancel(c.ctx)
	req := &h2Req{cancel: cancel, done: make(chan struct{}), length: n, since: now, replay: replay, packet: packet}
	c.pending[req] = struct{}{}
	if native != nil {
		native.originals[req] = struct{}{}
		c.trimHistory(now)
	}
	if replay {
		c.replayPending[req] = struct{}{}
		packet.attempts++
		packet.lastAttemptAt = now
		packet.pending = req
	}
	c.mu.Unlock()
	go c.runRequest(reqCtx, req, body)
	return true, nil
}

func (c *h2Channel) findPacket(body []byte) *replayPacket {
	for _, entry := range c.history {
		if bytes.Equal(entry.body, body) {
			return entry
		}
	}
	return nil
}

// rememberPacket keeps encrypted packets of the current auth key for replays
func (c *h2Channel) rememberPacket(body []byte, now time.Time) *replayPacket {
	key := body[:8]
	if bytes.Equal(key, make([]byte, 8)) {
		return nil
	}
	if !bytes.Equal(c.replayKey, key) {
		c.history = nil
		c.replayKey = append([]byte(nil), key...)
	}
	if len(body) > replayHistoryBytes {
		return nil
	}
	if packet := c.findPacket(body); packet != nil {
		return packet
	}
	packet := &replayPacket{sentAt: now, body: body, sequence: c.requests, originals: map[*h2Req]struct{}{}}
	c.history = append(c.history, packet)
	return packet
}

func (c *h2Channel) trimHistory(now time.Time) {
	retained := 0
	for _, entry := range c.history {
		retained += len(entry.body)
	}
	for len(c.history) > replayHistoryPackets || retained > replayHistoryBytes {
		victim := 0
		for i, entry := range c.history {
			keep := len(entry.originals) > 0 ||
				(entry.retired && entry.attempts < replayMaxAttempts && now.Sub(entry.sentAt) <= replayMaxAge)
			if !keep {
				victim = i
				break
			}
		}
		retained -= len(c.history[victim].body)
		c.history = append(c.history[:victim], c.history[victim+1:]...)
	}
}

func (c *h2Channel) markSent(req *h2Req) {
	c.mu.Lock()
	if _, ok := c.pending[req]; ok {
		req.sentAt = time.Now()
	}
	c.mu.Unlock()
}

func (c *h2Channel) markReceiving(req *h2Req, receiving bool) {
	c.mu.Lock()
	req.receiving = receiving
	if receiving {
		c.lastProgress = time.Now()
	}
	c.mu.Unlock()
}

func (c *h2Channel) progress() {
	c.mu.Lock()
	c.lastProgress = time.Now()
	c.mu.Unlock()
}

func (c *h2Channel) runRequest(ctx context.Context, req *h2Req, body []byte) {
	defer close(req.done)
	reply, err := c.lane.post(ctx, body, c, req)
	c.mu.Lock()
	c.lastProgress = time.Now()
	if err == nil && len(reply) > 0 && !c.closed {
		if p := req.packet; p != nil {
			id := reply[:min(24, len(reply))]
			if !bytes.Equal(id, p.replyID) {
				p.replyID = append([]byte(nil), id...)
				p.attempts = 0
			}
		}
		if len(c.queue) >= cap(c.queue) || c.replyBytes+len(reply) > h2MaxChannelBytes {
			err = errors.New("H2 client response queue exceeded bound")
		} else {
			c.replyBytes += len(reply)
			c.queue <- h2Item{data: reply}
		}
	}
	closed := c.closed
	c.mu.Unlock()
	if err != nil && !errors.Is(err, errH2Canceled) {
		stats.H2Errors.Add(1)
		var te *mtprotoTransportError
		if errors.As(err, &te) {
			if !closed {
				logx.Warnf("[%s] H2 lane=%d channel=%d host=%s MTProto transport error=%v bytes=%d replay=%v; reporting to client",
					c.label, c.lane.id, c.id, c.lane.host, te, len(body), req.replay)
			}
		} else if !closed {
			logx.Warnf("[%s] H2 lane=%d channel=%d request failed: %v; closing native channel, reconnect may select another domain",
				c.label, c.lane.id, c.id, err)
		}
		c.fail(err)
	}
	c.finishRequest(req)
}

func (c *h2Channel) finishRequest(req *h2Req) {
	c.mu.Lock()
	if _, ok := c.pending[req]; !ok {
		c.mu.Unlock()
		return
	}
	delete(c.pending, req)
	delete(c.replayPending, req)
	c.pendingBytes -= req.length
	packet := req.packet
	if packet == nil {
		for _, entry := range c.history {
			if _, ok := entry.originals[req]; ok || entry.pending == req {
				packet = entry
				break
			}
		}
	}
	if packet != nil {
		delete(packet.originals, req)
		if packet.pending == req {
			packet.pending = nil
		}
	}
	bump(&c.notify)
	c.wakeReceiverLocked()
	c.mu.Unlock()
	c.lane.release(req.length)
}

// recoverLoop sends replays as polls and frees capacity taken by overdue requests
func (c *h2Channel) recoverLoop() {
	ticker := time.NewTicker(replayCheck)
	defer ticker.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-c.wakeup:
		case <-ticker.C:
		}
		now := time.Now()
		c.makeUploadRoom(now)

		c.mu.Lock()
		if c.closed {
			c.mu.Unlock()
			return
		}
		if c.nativeBackpressure() {
			c.mu.Unlock()
			continue
		}
		candidates := append([]*replayPacket(nil), c.history...)
		overdue := map[*replayPacket]bool{}
		for _, e := range candidates {
			overdue[e] = c.overdueOriginal(e, now)
		}
		sort.SliceStable(candidates, func(i, j int) bool {
			a, b := candidates[i], candidates[j]
			if overdue[a] != overdue[b] {
				return overdue[a]
			}
			if overdue[a] {
				return a.sequence < b.sequence
			}
			return a.sequence > b.sequence
		})
		var chosen *replayPacket
		var replace *h2Req
		for _, entry := range candidates {
			if !c.replayDue(entry, now) {
				continue
			}
			replace = entry.pending
			if replace == nil && len(c.replayPending) >= maxChannelReplays {
				for _, req := range c.sortedPending() {
					if _, ok := c.replayPending[req]; ok && !req.receiving {
						replace = req
						break
					}
				}
				if replace == nil {
					continue
				}
			}
			if replace != nil && (replace.receiving || replace.sentAt.IsZero() || now.Sub(replace.sentAt) < replaySlot) {
				replace = nil
				continue
			}
			chosen = entry
			break
		}
		c.mu.Unlock()
		if chosen == nil {
			continue
		}
		if replace != nil {
			replace.cancel()
			<-replace.done
			c.mu.Lock()
			c.replayRotations++
			c.mu.Unlock()
		}
		if _, err := c.send(chosen.body, false, true); err != nil {
			return
		}
	}
}

func (c *h2Channel) receive() ([]byte, error) {
	item := <-c.queue
	if item.err != nil {
		return nil, item.err
	}
	c.mu.Lock()
	c.replyBytes -= len(item.data)
	c.wakeReceiverLocked()
	c.mu.Unlock()
	return item.data, nil
}

func (c *h2Channel) fail(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	var te *mtprotoTransportError
	if errors.As(err, &te) {
		c.transportErr = te
	}
	c.closed = true
	select {
	case c.wakeup <- struct{}{}:
	default:
	}
	for len(c.queue) > 0 {
		<-c.queue
	}
	c.replyBytes = 0
	c.queue <- h2Item{err: err}
	bump(&c.notify)
}

func (c *h2Channel) isClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

func (c *h2Channel) close() {
	c.closeOnce.Do(func() {
		c.fail(errH2ChannelClosed)
		c.stop()
		c.mu.Lock()
		reqs := c.sortedPending()
		c.mu.Unlock()
		for _, req := range reqs {
			req.cancel()
		}
		for _, req := range reqs {
			<-req.done
		}
		c.mu.Lock()
		c.history = nil
		c.replayPending = map[*h2Req]struct{}{}
		c.mu.Unlock()
		c.lane.removeChannel(c)
		logx.Debugf("[%s] H2 channel=%d closed lane=%d requests=%d replays=%d rotations=%d capacity_rotations=%d up=%d down=%d duration=%.1fs",
			c.label, c.id, c.lane.id, c.requests, c.replayRequests, c.replayRotations, c.capacityRotation,
			c.up, c.down, time.Since(c.started).Seconds())
	})
}

// h2Lane is one HTTP/2 client to one Cloudflare host shared by many channels
type h2Lane struct {
	host   string
	id     int
	client *http.Client
	tr     *http.Transport

	mu               sync.Mutex
	notify           chan struct{}
	channels         map[*h2Channel]struct{}
	inflight         int
	queuedBytes      int
	replyBufferBytes int
	failedUntil      time.Time
	closed           bool

	tcpConnections    atomic.Int64
	requests          atomic.Int64
	responses         atomic.Int64
	errors            atomic.Int64
	capacityRotations atomic.Int64
}

func newH2Lane(host string, id int) *h2Lane {
	tr := &http.Transport{
		Proxy:               nil,
		DialContext:         (&net.Dialer{Timeout: 8 * time.Second}).DialContext,
		TLSClientConfig:     &tls.Config{},
		TLSHandshakeTimeout: 8 * time.Second,
		ForceAttemptHTTP2:   true,
		DisableCompression:  true,
		IdleConnTimeout:     90 * time.Second,
		HTTP2: &http.HTTP2Config{
			MaxReceiveBufferPerStream:     256 * 1024,
			MaxReceiveBufferPerConnection: 4 * 1024 * 1024,
			SendPingTimeout:               20 * time.Second,
			PingTimeout:                   10 * time.Second,
		},
	}
	protocols := new(http.Protocols)
	protocols.SetHTTP2(true)
	tr.Protocols = protocols
	return &h2Lane{host: host, id: id, tr: tr, client: &http.Client{Transport: tr}, notify: make(chan struct{}),
		channels: map[*h2Channel]struct{}{}}
}

func (l *h2Lane) addChannel(c *h2Channel) {
	l.mu.Lock()
	l.channels[c] = struct{}{}
	l.mu.Unlock()
}

func (l *h2Lane) removeChannel(c *h2Channel) {
	l.mu.Lock()
	delete(l.channels, c)
	l.mu.Unlock()
}

func (l *h2Lane) channelList() []*h2Channel {
	l.mu.Lock()
	defer l.mu.Unlock()
	result := make([]*h2Channel, 0, len(l.channels))
	for c := range l.channels {
		result = append(result, c)
	}
	return result
}

func (l *h2Lane) hasCapacityLocked(n int) bool {
	return l.inflight < h2MaxLaneRequests && l.queuedBytes+n <= h2MaxLaneBytes
}

func (l *h2Lane) hasCapacity(n int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.hasCapacityLocked(n)
}

func (l *h2Lane) reserve(ctx context.Context, n int, wait bool) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !wait && !l.hasCapacityLocked(n) {
		return false, nil
	}
	if wait {
		if err := waitCond(ctx, &l.mu, &l.notify, func() bool { return l.closed || l.hasCapacityLocked(n) }, h2RequestTimeout); err != nil {
			if ctx.Err() != nil {
				return false, errH2ChannelClosed
			}
			return false, errors.New("H2 lane capacity wait timed out")
		}
	}
	if l.closed {
		return false, errH2LaneClosed
	}
	l.inflight++
	l.queuedBytes += n
	return true, nil
}

func (l *h2Lane) release(n int) {
	l.mu.Lock()
	l.inflight--
	l.queuedBytes -= n
	bump(&l.notify)
	l.mu.Unlock()
}

func (l *h2Lane) isFailed(now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return now.Before(l.failedUntil)
}

func (l *h2Lane) markFailed() {
	l.mu.Lock()
	l.failedUntil = time.Now().Add(h2DomainCooldown)
	l.mu.Unlock()
}

func (l *h2Lane) trace() *httptrace.ClientTrace {
	return &httptrace.ClientTrace{
		ConnectStart: func(network, addr string) {
			l.tcpConnections.Add(1)
			stats.H2TCPConnections.Add(1)
			logx.Debugf("H2 lane=%d TCP_OPEN connection=%d host=%s", l.id, l.tcpConnections.Load(), l.host)
		},
	}
}

// preflight checks that the host answers /api over HTTP/2
func (l *h2Lane) preflight(ctx context.Context) error {
	req, err := http.NewRequestWithContext(httptrace.WithClientTrace(ctx, l.trace()), http.MethodHead, "https://"+l.host+"/api", nil)
	if err != nil {
		return err
	}
	started := time.Now()
	resp, err := l.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.ProtoMajor != 2 {
		return fmt.Errorf("CF /api preflight negotiated %s", resp.Proto)
	}
	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) && resp.StatusCode != 405 && resp.StatusCode != 501 {
		return fmt.Errorf("CF /api preflight HTTP %d cf_ray=%s", resp.StatusCode, resp.Header.Get("cf-ray"))
	}
	logx.Debugf("H2 lane=%d ready host=%s status=%d ms=%d", l.id, l.host, resp.StatusCode, time.Since(started).Milliseconds())
	return nil
}

// post sends one MTProto packet and returns the answer of Telegram
func (l *h2Lane) post(ctx context.Context, body []byte, ch *h2Channel, req *h2Req) ([]byte, error) {
	l.requests.Add(1)
	stats.H2Requests.Add(1)
	started := time.Now()
	reqCtx, cancel := context.WithTimeout(ctx, h2RequestTimeout)
	defer cancel()
	trace := l.trace()
	trace.WroteRequest = func(httptrace.WroteRequestInfo) { ch.markSent(req) }
	httpReq, err := http.NewRequestWithContext(httptrace.WithClientTrace(reqCtx, trace), http.MethodPost,
		"https://"+l.host+"/api", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/octet-stream")
	httpReq.Header.Set("Accept-Encoding", "identity")

	buffered := 0
	defer func() {
		if buffered > 0 {
			l.mu.Lock()
			l.replyBufferBytes -= buffered
			l.mu.Unlock()
		}
	}()
	fail := func(err error) ([]byte, error) {
		if ctx.Err() != nil {
			return nil, errH2Canceled
		}
		l.errors.Add(1)
		l.markFailed()
		return nil, err
	}

	resp, err := l.client.Do(httpReq)
	if err != nil {
		return fail(err)
	}
	defer resp.Body.Close()
	ray := resp.Header.Get("cf-ray")
	if resp.ProtoMajor != 2 {
		return fail(fmt.Errorf("CF /api downgraded to %s", resp.Proto))
	}
	switch resp.StatusCode {
	case 403, 404, 429, 444:
		l.errors.Add(1)
		return nil, &mtprotoTransportError{code: int32(-resp.StatusCode), source: fmt.Sprintf("HTTP %d cf_ray=%s", resp.StatusCode, ray)}
	case 200:
	default:
		return fail(fmt.Errorf("CF /api HTTP %d cf_ray=%s", resp.StatusCode, ray))
	}
	if strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "text/html") {
		return fail(errors.New("CF /api returned HTML instead of MTProto"))
	}
	headersAt := time.Now()
	ch.markReceiving(req, true)
	defer ch.markReceiving(req, false)

	declared := -1
	if v := resp.Header.Get("Content-Length"); v != "" {
		declared, err = strconv.Atoi(v)
		if err != nil || declared < 0 || declared > h2MaxPacket {
			return fail(errors.New("H2 response Content-Length exceeds native packet bound"))
		}
	}
	content := make([]byte, 0, max(declared, 0))
	buf := make([]byte, 32*1024)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if len(content)+n > h2MaxPacket {
				return fail(errors.New("H2 response exceeds native packet bound"))
			}
			l.mu.Lock()
			if l.replyBufferBytes+n > h2MaxLaneReplyBytes {
				l.mu.Unlock()
				return nil, errReplyBufferFull
			}
			l.replyBufferBytes += n
			l.mu.Unlock()
			buffered += n
			content = append(content, buf[:n]...)
			ch.progress()
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return fail(rerr)
		}
	}
	if declared >= 0 && len(content) != declared {
		return fail(errors.New("incomplete H2 response body"))
	}
	if len(content)%4 != 0 {
		return fail(errors.New("CF /api response is not an aligned MTProto packet"))
	}
	if len(content) == 4 {
		if code := int32(binary.LittleEndian.Uint32(content)); code < 0 {
			l.errors.Add(1)
			return nil, &mtprotoTransportError{code: code, source: "HTTP 200 payload"}
		}
	}
	l.responses.Add(1)
	if total := time.Since(started); total >= h2SlowResponseSeconds {
		logx.Debugf("H2 SLOW lane=%d channel=%d bytes=%d replay=%v headers_ms=%d body_ms=%d",
			l.id, ch.id, len(content), req.replay, headersAt.Sub(started).Milliseconds(), time.Since(headersAt).Milliseconds())
	}
	return content, nil
}

func (l *h2Lane) close() {
	l.mu.Lock()
	l.closed = true
	bump(&l.notify)
	l.mu.Unlock()
	for _, c := range l.channelList() {
		c.close()
	}
	l.tr.CloseIdleConnections()
	logx.Debugf("H2 CLOSED lane=%d host=%s tcp_connections=%d requests=%d replies=%d errors=%d",
		l.id, l.host, l.tcpConnections.Load(), l.requests.Load(), l.responses.Load(), l.errors.Load())
}

// h2Pool keeps lanes per Cloudflare host
type h2Pool struct {
	s               *Server
	mu              sync.Mutex
	lanes           map[string]*h2Lane
	hostLocks       map[string]*sync.Mutex
	failedUntil     map[string]time.Time
	setupRetryAfter map[int]time.Time
	nextLane        int
	nextChannel     int
	closed          bool
}

func newH2Pool(s *Server) *h2Pool {
	return &h2Pool{s: s, lanes: map[string]*h2Lane{}, hostLocks: map[string]*sync.Mutex{},
		failedUntil: map[string]time.Time{}, setupRetryAfter: map[int]time.Time{}}
}

// open returns a channel for a media connection of DC, nil if HTTP/2 isn't available now
func (p *h2Pool) open(ctx context.Context, dc int, label string) *h2Channel {
	p.mu.Lock()
	if p.closed || time.Now().Before(p.setupRetryAfter[dc]) {
		p.mu.Unlock()
		return nil
	}
	p.mu.Unlock()
	setupCtx, cancel := context.WithTimeout(ctx, h2SetupTimeout)
	defer cancel()
	ch := p.doOpen(setupCtx, dc, label)
	if ch == nil && setupCtx.Err() == context.DeadlineExceeded && ctx.Err() == nil {
		p.mu.Lock()
		p.setupRetryAfter[dc] = time.Now().Add(h2DomainCooldown)
		p.mu.Unlock()
		stats.H2Errors.Add(1)
		logx.Warnf("[%s] H2 DC%d setup timed out; using WS for %.0fs", label, dc, h2DomainCooldown.Seconds())
	}
	return ch
}

func (p *h2Pool) doOpen(ctx context.Context, dc int, label string) *h2Channel {
	if _, ok := DCDefaultIPs[dc]; !ok {
		logx.Warnf("[%s] H2 unsupported DC%d -> existing WS route", label, dc)
		return nil
	}
	for _, base := range p.s.balancer.domainsForDC(dc) {
		if ctx.Err() != nil {
			return nil
		}
		if p.s.cf.tlsInCooldown(base) {
			continue
		}
		host := fmt.Sprintf("kws%d.%s", dc, base)
		p.mu.Lock()
		lock, ok := p.hostLocks[host]
		if !ok {
			lock = &sync.Mutex{}
			p.hostLocks[host] = lock
		}
		p.mu.Unlock()

		lock.Lock()
		ch := p.openOnHost(ctx, dc, base, host, label)
		lock.Unlock()
		if ch != nil {
			return ch
		}
	}
	logx.Debugf("[%s] H2 DC%d unavailable; using WS", label, dc)
	return nil
}

func (p *h2Pool) openOnHost(ctx context.Context, dc int, base, host, label string) *h2Channel {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	lane := p.lanes[host]
	now := time.Now()
	if now.Before(p.failedUntil[host]) || (lane != nil && lane.isFailed(now)) {
		p.mu.Unlock()
		return nil
	}
	if lane == nil {
		p.nextLane++
		lane = newH2Lane(host, p.nextLane)
		p.mu.Unlock()
		if err := lane.preflight(ctx); err != nil {
			lane.close()
			if ctx.Err() != nil {
				return nil
			}
			stats.H2Errors.Add(1)
			p.mu.Lock()
			p.failedUntil[host] = time.Now().Add(h2DomainCooldown)
			p.mu.Unlock()
			logx.Warnf("[%s] H2 lane=%d preflight failed host=%s: %v", label, lane.id, host, err)
			return nil
		}
		p.mu.Lock()
		if p.closed {
			p.mu.Unlock()
			lane.close()
			return nil
		}
		p.lanes[host] = lane
	}
	p.nextChannel++
	id := p.nextChannel
	p.mu.Unlock()
	ch := newH2Channel(lane, id, label)
	p.s.balancer.updateDomainForDC(dc, base)
	logx.Infof("[%s] ROUTE DC%d media -> H2 lane=%d channel=%d host=%s channels=%d tcp_connections=%d",
		label, dc, lane.id, id, host, len(lane.channelList()), lane.tcpConnections.Load())
	return ch
}

func (p *h2Pool) close() {
	p.mu.Lock()
	p.closed = true
	lanes := make([]*h2Lane, 0, len(p.lanes))
	for _, l := range p.lanes {
		lanes = append(lanes, l)
	}
	p.lanes = map[string]*h2Lane{}
	p.mu.Unlock()
	for _, l := range lanes {
		l.close()
	}
}

// bridgeH2 connects the native client with an HTTP/2 channel
func (s *Server) bridgeH2(cl *clientConn, ch *h2Channel, ctx *cryptoCtx, tag [4]byte) {
	writeNative := func(data []byte) error {
		ctx.cltEnc.XORKeyStream(data, data)
		_, err := cl.w.Write(data)
		return err
	}
	errc := make(chan error, 3)
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		for {
			body, quick, err := readNativePacket(cl.r, ctx.cltDec, tag)
			if err != nil {
				errc <- fmt.Errorf("native: %w", err)
				return
			}
			cl.countUp(len(body))
			if _, err := ch.send(body, quick, false); err != nil {
				errc <- err
				return
			}
		}
	}()
	go func() {
		defer wg.Done()
		for {
			body, err := ch.receive()
			if err != nil {
				errc <- err
				return
			}
			if ch.isClosed() {
				errc <- errors.New("native: H2 channel closed before native response")
				return
			}
			ch.mu.Lock()
			ch.deliveringSince = time.Now()
			ch.mu.Unlock()
			frame, err := encodeReply(body, tag)
			if err == nil {
				err = writeNative(frame)
			}
			ch.mu.Lock()
			ch.deliveringSince = time.Time{}
			if err == nil {
				ch.down += len(body)
				ch.lastProgress = time.Now()
				ch.wakeReceiverLocked()
			}
			ch.mu.Unlock()
			if err != nil {
				errc <- fmt.Errorf("native: %w", err)
				return
			}
			cl.countDown(len(body))
		}
	}()
	go func() {
		defer wg.Done()
		ch.recoverLoop()
		errc <- errH2ChannelClosed
	}()

	first := <-errc
	ch.fail(errH2ChannelClosed)
	ch.stop()
	cl.interrupt()
	wg.Wait()

	ch.mu.Lock()
	te := ch.transportErr
	ch.mu.Unlock()
	if te != nil {
		frame, err := encodeReply(binary.LittleEndian.AppendUint32(nil, uint32(te.code)), tag)
		if err == nil {
			writeNative(frame)
			cl.countDown(4)
		}
	} else if first != nil && !errors.Is(first, errH2ChannelClosed) && !strings.HasPrefix(first.Error(), "native:") {
		logx.Warnf("[%s] H2 channel=%d bridge ended: %v", ch.label, ch.id, first)
	} else {
		logx.Debugf("[%s] H2 channel=%d native closed: %v", ch.label, ch.id, first)
	}
	ch.close()
}
