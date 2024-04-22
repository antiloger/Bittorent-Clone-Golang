package p2p

import (
	"log"

	"github.com/antiloger/bittorrent-clone-golang/client"
	"github.com/antiloger/bittorrent-clone-golang/peers"
)

type Torrent struct {
	Peers       []peers.Peer
	PeerID      [20]byte
	InfoHash    [20]byte
	PieceHashes [][20]byte
	PieceLength int
	Length      int
	Name        string
}

type pieceWork struct {
	index  int
	hash   [20]byte
	length int
}

type pieceResult struct {
	index int
	buf   []byte
}

type pieceProgress struct {
	index      int
	client     *client.Client
	buf        []byte
	downloaded int
	requested  int
	backlog    int
}

func (t *Torrent) startDownloadWorker(peer peers.Peer, workq chan *pieceWork, results chan *pieceResult) {
	c, err := client.New(peer, t.PeerID, t.InfoHash)
	if err != nil {
		log.Printf("completed handshake with %s. Disconnecting\n", peer.IP)
	}

	defer c.Conn.Close()

	log.Printf("completed handshake with %s\n", peer.IP)

	c.SendUnchoke()
	c.SendInterested()

	for pw := range workq {
		if !c.Bitfield.HasPiece(pw.index) {
			workq <- pw
			continue
		}

		buf, err := attemptDownloadPiece(c, pw)
		if err != nil {
			log.Printf("failed to download piece %d from %s\n", pw.index, peer.IP)
			workq <- pw
			continue
		}

		c.SendHave(pw.index)
		results <- &pieceResult{pw.index, buf}
	}
}

func (t *Torrent) Download() {
	log.Println("Starting Dowload ...")

	workQ := make(chan *pieceWork, len(t.PieceHashes))
	results := make(chan *pieceResult)

	for index, hash := range t.PieceHashes {
		length := t.calculatePieceSize(index)
		workQ <- &pieceWork{index, hash, length}
	}

	for _, peer := range t.Peers {
		go t.startDownloadWorker(peer, workQ, results)
	}

	buf := make([]byte, t.Length)
	donePieces := 0
	for donePieces < len(t.PieceHashes) {
		res := <-results
		begin, end := t.calculateBoundsForPiece(res.index)
		copy(buf[begin:end], res.buf)
		donePieces++
	}
	close(workQ)
}
