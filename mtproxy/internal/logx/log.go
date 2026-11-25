// Package logx is the logger of the proxy: levels, console, a rotated file
// and a ring buffer that the web interface reads.
// Domains other than telegram.org are partly hidden, like in tg-ws-proxy.
package logx

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Level int32

const (
	Debug Level = iota
	Info
	Warn
	Error
)

func (l Level) String() string {
	switch l {
	case Debug:
		return "DEBUG"
	case Info:
		return "INFO"
	case Warn:
		return "WARN"
	default:
		return "ERROR"
	}
}

// Entry is one line of the log
type Entry struct {
	Seq   uint64    `json:"seq"`
	Time  time.Time `json:"time"`
	Level string    `json:"level"`
	Msg   string    `json:"msg"`
}

const ringSize = 3000

type logger struct {
	level atomic.Int32

	mu       sync.Mutex
	console  io.Writer
	file     *os.File
	filePath string
	fileSize int64
	maxSize  int64
	ring     []Entry
	next     int // position for the next entry in ring
	seq      uint64
}

var std = newLogger()

func newLogger() *logger {
	l := &logger{console: os.Stderr, ring: make([]Entry, 0, ringSize)}
	l.level.Store(int32(Info))
	return l
}

// SetLevel sets the minimal written level
func SetLevel(level Level) { std.level.Store(int32(level)) }

// SetVerbose switches between Debug and Info levels
func SetVerbose(verbose bool) {
	if verbose {
		SetLevel(Debug)
	} else {
		SetLevel(Info)
	}
}

// Enabled reports whether messages of level are written
func Enabled(level Level) bool { return int32(level) >= std.level.Load() }

// SetFile starts writing the log to path (empty path stops it). When the file grows over maxMB
// it is renamed to path.1, the previous path.1 is deleted
func SetFile(path string, maxMB float64) error {
	std.mu.Lock()
	defer std.mu.Unlock()
	if std.file != nil && std.filePath == path {
		std.maxSize = int64(maxMB * 1024 * 1024)
		return nil
	}
	if std.file != nil {
		std.file.Close()
		std.file = nil
	}
	std.filePath = path
	std.maxSize = int64(maxMB * 1024 * 1024)
	if path == "" {
		return nil
	}
	return std.openFile()
}

// FilePath returns the path of the log file, empty if it isn't written
func FilePath() string {
	std.mu.Lock()
	defer std.mu.Unlock()
	return std.filePath
}

func (l *logger) openFile() error {
	f, err := os.OpenFile(l.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	info, err := f.Stat()
	if err == nil {
		l.fileSize = info.Size()
	}
	l.file = f
	return nil
}

func (l *logger) rotate() {
	l.file.Close()
	l.file = nil
	os.Remove(l.filePath + ".1")
	os.Rename(l.filePath, l.filePath+".1")
	if err := l.openFile(); err != nil {
		fmt.Fprintln(l.console, "can't reopen log file:", err)
	}
}

func (l *logger) write(level Level, msg string) {
	if int32(level) < l.level.Load() {
		return
	}
	msg = Censor(msg)
	now := time.Now()
	line := fmt.Sprintf("%s  %-5s  %s\n", now.Format("2006-01-02 15:04:05"), level, msg)

	l.mu.Lock()
	defer l.mu.Unlock()
	l.seq++
	entry := Entry{Seq: l.seq, Time: now, Level: level.String(), Msg: msg}
	if len(l.ring) < ringSize {
		l.ring = append(l.ring, entry)
	} else {
		l.ring[l.next] = entry
	}
	l.next = (l.next + 1) % ringSize

	io.WriteString(l.console, line)
	if l.file != nil {
		n, _ := io.WriteString(l.file, line)
		l.fileSize += int64(n)
		if l.maxSize > 0 && l.fileSize > l.maxSize {
			l.rotate()
		}
	}
}

// Since returns entries with Seq greater than seq (at most limit of the latest ones)
func Since(seq uint64, limit int) []Entry {
	std.mu.Lock()
	defer std.mu.Unlock()
	result := make([]Entry, 0, 64)
	count := len(std.ring)
	start := 0
	if count == ringSize {
		start = std.next
	}
	for i := 0; i < count; i++ {
		entry := std.ring[(start+i)%ringSize]
		if entry.Seq > seq {
			result = append(result, entry)
		}
	}
	if limit > 0 && len(result) > limit {
		result = result[len(result)-limit:]
	}
	return result
}

func Debugf(format string, args ...interface{}) {
	if Enabled(Debug) {
		std.write(Debug, fmt.Sprintf(format, args...))
	}
}

func Infof(format string, args ...interface{})  { std.write(Info, fmt.Sprintf(format, args...)) }
func Warnf(format string, args ...interface{})  { std.write(Warn, fmt.Sprintf(format, args...)) }
func Errorf(format string, args ...interface{}) { std.write(Error, fmt.Sprintf(format, args...)) }

var domainPattern = regexp.MustCompile(`(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}`)

func isWordByte(b byte) bool {
	return b == '-' || b == '_' || (b >= '0' && b <= '9') || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// Censor hides the second half of every label of domains except telegram.org ones,
// so the log can be shared without the user's own domains
func Censor(s string) string {
	matches := domainPattern.FindAllStringIndex(s, -1)
	if matches == nil {
		return s
	}
	var b strings.Builder
	last := 0
	for _, m := range matches {
		start, end := m[0], m[1]
		if (start > 0 && isWordByte(s[start-1])) || (end < len(s) && isWordByte(s[end])) {
			continue
		}
		domain := s[start:end]
		b.WriteString(s[last:start])
		b.WriteString(censorDomain(domain))
		last = end
	}
	b.WriteString(s[last:])
	return b.String()
}

func censorDomain(domain string) string {
	lower := strings.ToLower(strings.TrimSuffix(domain, "."))
	if lower == "telegram.org" || strings.HasSuffix(lower, ".telegram.org") ||
		strings.HasSuffix(lower, ".log") || strings.HasSuffix(lower, ".json") || strings.HasSuffix(lower, ".go") {
		return domain
	}
	parts := strings.Split(domain, ".")
	for i := 0; i < len(parts)-1; i++ {
		keep := len(parts[i]) / 2
		parts[i] = parts[i][:keep] + strings.Repeat("*", len(parts[i])-keep)
	}
	return strings.Join(parts, ".")
}
