package proxy

import "encoding/binary"

// msgSplitter splits the re-encrypted TCP stream into MTProto transport packets,
// so each of them is sent in its own WebSocket frame
type msgSplitter struct {
	proto    uint32
	cipher   []byte
	plain    []byte
	disabled bool
}

func newMsgSplitter(proto uint32) *msgSplitter { return &msgSplitter{proto: proto} }

// split takes the encrypted chunk and its plaintext, returns complete encrypted packets
func (s *msgSplitter) split(chunk, plain []byte) [][]byte {
	if len(chunk) == 0 {
		return nil
	}
	if s.disabled {
		return [][]byte{chunk}
	}
	s.cipher = append(s.cipher, chunk...)
	s.plain = append(s.plain, plain...)

	var parts [][]byte
	offset := 0
	total := len(s.cipher)
	for offset < total {
		packetLen := s.nextPacketLen(offset, total-offset)
		if packetLen < 0 {
			break
		}
		if packetLen == 0 {
			parts = append(parts, append([]byte(nil), s.cipher[offset:]...))
			offset = total
			s.disabled = true
			break
		}
		parts = append(parts, append([]byte(nil), s.cipher[offset:offset+packetLen]...))
		offset += packetLen
	}
	if offset > 0 {
		s.cipher = append(s.cipher[:0], s.cipher[offset:]...)
		s.plain = append(s.plain[:0], s.plain[offset:]...)
	}
	return parts
}

// flush returns the incomplete tail
func (s *msgSplitter) flush() []byte {
	if len(s.cipher) == 0 {
		return nil
	}
	tail := append([]byte(nil), s.cipher...)
	s.cipher = s.cipher[:0]
	s.plain = s.plain[:0]
	return tail
}

// nextPacketLen returns the length of the packet at offset, -1 if more data is needed,
// 0 if the stream can't be split (then splitting is disabled)
func (s *msgSplitter) nextPacketLen(offset, avail int) int {
	if avail <= 0 {
		return -1
	}
	switch s.proto {
	case protoAbridged:
		first := s.plain[offset]
		var payloadLen, headerLen int
		if first == 0x7F || first == 0xFF {
			if avail < 4 {
				return -1
			}
			b := s.plain[offset+1 : offset+4]
			payloadLen = (int(b[0]) | int(b[1])<<8 | int(b[2])<<16) * 4
			headerLen = 4
		} else {
			payloadLen = int(first&0x7F) * 4
			headerLen = 1
		}
		if payloadLen <= 0 {
			return 0
		}
		if avail < headerLen+payloadLen {
			return -1
		}
		return headerLen + payloadLen
	case protoIntermediate, protoPadded:
		if avail < 4 {
			return -1
		}
		payloadLen := int(binary.LittleEndian.Uint32(s.plain[offset:offset+4]) & 0x7FFFFFFF)
		if payloadLen <= 0 {
			return 0
		}
		if avail < 4+payloadLen {
			return -1
		}
		return 4 + payloadLen
	}
	return 0
}
