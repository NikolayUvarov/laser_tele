package proxy

import "fmt"

const (
	handshakeLen = 64
	skipLen      = 8
	prekeyLen    = 32
	keyLen       = 32
	ivLen        = 16
	protoTagPos  = 56
	dcIdxPos     = 60

	protoAbridged     uint32 = 0xEFEFEFEF
	protoIntermediate uint32 = 0xEEEEEEEE
	protoPadded       uint32 = 0xDDDDDDDD

	wsPath     = "/apiws"
	wsPathTest = "/apiws_test"
)

var (
	tagAbridged     = [4]byte{0xef, 0xef, 0xef, 0xef}
	tagIntermediate = [4]byte{0xee, 0xee, 0xee, 0xee}
	tagPadded       = [4]byte{0xdd, 0xdd, 0xdd, 0xdd}
)

// starts of the init packet that Telegram servers treat as other protocols
var reservedStarts = [][4]byte{
	{'H', 'E', 'A', 'D'}, {'P', 'O', 'S', 'T'}, {'G', 'E', 'T', ' '}, {0xee, 0xee, 0xee, 0xee},
	{0xdd, 0xdd, 0xdd, 0xdd}, {0x16, 0x03, 0x01, 0x02}, {'O', 'P', 'T', 'I'},
}

// DCDefaultIPs are addresses of production datacenters for the TCP fallback and CF worker
var DCDefaultIPs = map[int]string{
	1:   "149.154.175.50",
	2:   "149.154.167.51",
	3:   "149.154.175.100",
	4:   "149.154.167.91",
	5:   "149.154.171.5",
	203: "91.105.192.100",
}

// DCTestIPs are addresses of test datacenters
var DCTestIPs = map[int]string{
	1: "149.154.175.10",
	2: "149.154.167.40",
	3: "149.154.175.117",
}

// TestDCs are the DCs checked by the connectivity tests
var TestDCs = []int{1, 2, 3, 4, 5, 203}

func wsDomains(dc int, media bool) []string {
	if dc == 203 {
		dc = 2
	}
	if !media {
		return []string{fmt.Sprintf("kws%d.web.telegram.org", dc)}
	}
	return []string{fmt.Sprintf("kws%d-1.web.telegram.org", dc), fmt.Sprintf("kws%d.web.telegram.org", dc)}
}

func protoOfTag(tag [4]byte) uint32 {
	switch tag {
	case tagAbridged:
		return protoAbridged
	case tagIntermediate:
		return protoIntermediate
	default:
		return protoPadded
	}
}

// HumanBytes formats n like 1.5MB
func HumanBytes(n int64) string {
	value := float64(n)
	for _, unit := range []string{"B", "KB", "MB", "GB"} {
		if value < 1024 && value > -1024 {
			return fmt.Sprintf("%.1f%s", value, unit)
		}
		value /= 1024
	}
	return fmt.Sprintf("%.1fTB", value)
}

func mediaTag(media bool) string {
	if media {
		return " media"
	}
	return ""
}
