package handshake

type Handshake struct {
	Pstr     string
	InfoHash [20]byte
	PeerID   [20]byte
}

func (h *Handshake) Serialize() []byte {
	buf := make([]byte, len(h.Pstr)+49)
	buf[0] = byte(len(h.Pstr))
	cur := 1

	cur += copy(buf[cur:], h.Pstr)
	cur += copy(buf[cur:], make([]byte, 8))
	cur += copy(buf[cur:], h.InfoHash[:])
	cur += copy(buf[cur:], h.PeerID[:])
	return buf
}
