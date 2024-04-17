package client

import (
	"net"
	"time"

	"github.com/antiloger/bittorrent-clone-golang/peers"
)

func New(peer peers.Peer) {
	conn, err := net.DialTimeout("tcp", peer.String(), 3*time.Second)
	if err != nil {
		return nil, err
	}
}
