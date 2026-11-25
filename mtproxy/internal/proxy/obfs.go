package proxy

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
)

// cryptoCtx re-encrypts the stream: data of the client is decrypted with its key
// and encrypted with the key of the relay connection to Telegram, and back
type cryptoCtx struct {
	cltDec cipher.Stream // decrypt data from the client
	cltEnc cipher.Stream // encrypt data to the client
	tgEnc  cipher.Stream // encrypt data to Telegram
	tgDec  cipher.Stream // decrypt data from Telegram
}

func newCTR(key, iv []byte) cipher.Stream {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err) // keys always have the right length
	}
	return cipher.NewCTR(block, iv)
}

func skipKeystream(stream cipher.Stream, n int) {
	buf := make([]byte, n)
	stream.XORKeyStream(buf, buf)
}

func reversed(b []byte) []byte {
	r := make([]byte, len(b))
	for i := range b {
		r[len(b)-1-i] = b[i]
	}
	return r
}

type handshakeInfo struct {
	dc       int // absolute DC number from the init packet
	media    bool
	tag      [4]byte
	prekeyIV []byte // 48 bytes of the init packet after the first 8
	secret   []byte
}

// tryHandshake decrypts the 64-byte obfuscated2 init with secret and checks the protocol tag
func tryHandshake(handshake, secret []byte) (handshakeInfo, bool) {
	prekeyIV := append([]byte(nil), handshake[skipLen:skipLen+prekeyLen+ivLen]...)
	key := sha256.Sum256(append(append([]byte(nil), prekeyIV[:prekeyLen]...), secret...))
	decrypted := make([]byte, handshakeLen)
	newCTR(key[:], prekeyIV[prekeyLen:]).XORKeyStream(decrypted, handshake[:handshakeLen])

	var tag [4]byte
	copy(tag[:], decrypted[protoTagPos:protoTagPos+4])
	if tag != tagAbridged && tag != tagIntermediate && tag != tagPadded {
		return handshakeInfo{}, false
	}
	dcIdx := int16(binary.LittleEndian.Uint16(decrypted[dcIdxPos : dcIdxPos+2]))
	dc := int(dcIdx)
	if dc < 0 {
		dc = -dc
	}
	return handshakeInfo{dc: dc, media: dcIdx < 0, tag: tag, prekeyIV: prekeyIV, secret: secret}, true
}

// generateRelayInit makes the init packet of the connection to Telegram with plain obfuscation
func generateRelayInit(tag [4]byte, dcIdx int16) []byte {
	rnd := make([]byte, handshakeLen)
	for {
		rand.Read(rnd)
		if rnd[0] == 0xEF {
			continue
		}
		var start [4]byte
		copy(start[:], rnd[:4])
		reserved := false
		for _, r := range reservedStarts {
			if start == r {
				reserved = true
				break
			}
		}
		if reserved {
			continue
		}
		if rnd[4] == 0 && rnd[5] == 0 && rnd[6] == 0 && rnd[7] == 0 {
			continue
		}
		break
	}

	key := rnd[skipLen : skipLen+prekeyLen]
	iv := rnd[skipLen+prekeyLen : skipLen+prekeyLen+ivLen]
	keystream := make([]byte, handshakeLen)
	newCTR(key, iv).XORKeyStream(keystream, keystream)

	tail := make([]byte, 8)
	copy(tail, tag[:])
	binary.LittleEndian.PutUint16(tail[4:6], uint16(dcIdx))
	rand.Read(tail[6:8])

	result := append([]byte(nil), rnd...)
	for i := 0; i < 8; i++ {
		result[protoTagPos+i] = tail[i] ^ keystream[protoTagPos+i]
	}
	return result
}

func buildCryptoCtx(prekeyIV, secret, relayInit []byte) *cryptoCtx {
	decKey := sha256.Sum256(append(append([]byte(nil), prekeyIV[:prekeyLen]...), secret...))
	cltDec := newCTR(decKey[:], prekeyIV[prekeyLen:])
	skipKeystream(cltDec, handshakeLen)

	encPrekeyIV := reversed(prekeyIV)
	encKey := sha256.Sum256(append(append([]byte(nil), encPrekeyIV[:prekeyLen]...), secret...))
	cltEnc := newCTR(encKey[:], encPrekeyIV[prekeyLen:])

	relayKey := relayInit[skipLen : skipLen+prekeyLen]
	relayIV := relayInit[skipLen+prekeyLen : skipLen+prekeyLen+ivLen]
	tgEnc := newCTR(relayKey, relayIV)
	skipKeystream(tgEnc, handshakeLen)

	relayDec := reversed(relayInit[skipLen : skipLen+prekeyLen+ivLen])
	tgDec := newCTR(relayDec[:keyLen], relayDec[keyLen:])

	return &cryptoCtx{cltDec: cltDec, cltEnc: cltEnc, tgEnc: tgEnc, tgDec: tgDec}
}
