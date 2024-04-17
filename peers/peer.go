package peers

import (
	"encoding/binary"
	"fmt"
	"net"
)

type Peer struct {
	IP   net.IP
	Port uint16
}

func Unmarshal(peersBin []byte) ([]Peer, error) {
	const peerSize = 6
	numofpeers := len(peersBin) / peerSize
	if len(peersBin)%peerSize != 0 {
		err := fmt.Errorf("received malformed peers")
		return nil, err
	}
	peers := make([]Peer, numofpeers)
	for i := 0; i < numofpeers; i++ {
		start := i * peerSize
		peers[i].IP = net.IP(peersBin[start : start+4])
		peers[i].Port = binary.BigEndian.Uint16(peersBin[start+4 : start+6])
	}
	return peers, nil
}
