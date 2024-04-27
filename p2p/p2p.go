package p2p

import (
	"log"
	"runtime"
	"time"

	"github.com/antiloger/bittorrent-clone-golang/client"
	"github.com/antiloger/bittorrent-clone-golang/message"
	"github.com/antiloger/bittorrent-clone-golang/peers"
)

const (
	MaxBlockSize = 16384
	MaxBacklog   = 5
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

func (state *pieceProgress) readMessage() error {
	msg, err := state.client.Read()
	if err != nil {
		return err
	}

	if msg == nil {
		return nil
	}

	switch msg.ID {
	case message.MsgUnchoke:
		state.client.Choked = false
	case message.MsgChoke:
		state.client.Choked = true
	case message.MsgHave:
		index, err := message.ParseHave(msg)
		if err != nil {
			return err
		}
		state.client.Bitfield.SetPiece(index)
	case message.MsgPiece:
		n, err := message.ParsePiece(state.index, state.buf, msg)
		if err != nil {
			return err
		}
		state.downloaded += n
		state.backlog--
	}
	return nil
}

func attemptDownloadPiece(c *client.Client, pw *pieceWork) ([]byte, error) {
	state := pieceProgress{
		index:  pw.index,
		client: c,
		buf:    make([]byte, pw.length),
	}

	c.Conn.SetDeadline(time.Now().Add(30 * time.Second))
	defer c.Conn.SetDeadline(time.Time{})
	for state.downloaded < pw.length {
		if !state.client.Choked {
			for state.backlog < MaxBacklog && state.requested < pw.length {
				blockSize := MaxBlockSize
				if pw.length-state.requested < blockSize {
					blockSize = pw.length - state.requested
				}

				err := c.SendRequest(pw.index, state.requested, blockSize)
				if err != nil {
					return nil, err
				}

				state.backlog++
				state.requested += blockSize
			}
		}

		err := state.readMessage()
		if err != nil {
			return nil, err
		}
	}
	return state.buf, nil
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

func (t *Torrent) calculateBoundsForPiece(index int) (begin int, end int) {
	begin = index * t.PieceLength
	end = begin + t.PieceLength
	if end > t.Length {
		end = t.Length
	}
	return begin, end
}

func (t *Torrent) calculatePieceSize(index int) int {
	begin, end := t.calculateBoundsForPiece(index)
	return end - begin
}

func (t *Torrent) Download() ([]byte, error) {
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

		percent := float64(donePieces) / float64(len(t.PieceHashes)) * 100
		numWorkers := runtime.NumGoroutine() - 1
		log.Printf("Downloaded piece %d from %d peers (%.2f%%)\n", donePieces, numWorkers, percent)
	}
	close(workQ)

	return buf, nil
}
