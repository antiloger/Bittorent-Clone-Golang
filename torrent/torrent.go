package torrent

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/antiloger/bittorrent-clone-golang/p2p"
	"github.com/antiloger/bittorrent-clone-golang/peers"
	"github.com/jackpal/bencode-go"
)

const Port uint16 = 6881

type bencodeInfo struct {
	Pieces      string `bencode:"pieces"`
	PieceLength int    `bencode:"piece length"`
	Length      int    `bencode:"length"`
	Name        string `bencode:"name"`
}

type bencodeTorrent struct {
	Announce string      `bencode:"announce"`
	Info     bencodeInfo `bencode:"info"`
}

type TorrentFile struct {
	Announce    string
	InfoHash    [20]byte
	PieceHashes [][20]byte
	PieceLength int
	Length      int
	Name        string
}

type bencodeTrackerResponse struct {
	Interval int    `bencode:"interval"`
	Peers    string `bencode:"peers"`
}

func (t *TorrentFile) requestPeers(peerID [20]byte, port uint16) ([]peers.Peer, error) {
	url, err := t.BuildTrackerURL(peerID, port)
	if err != nil {
		return nil, err
	}

	c := &http.Client{Timeout: 15 * time.Second}
	resp, err := c.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	trackerResp := bencodeTrackerResponse{}
	err = bencode.Unmarshal(resp.Body, &trackerResp)
	if err != nil {
		return nil, err
	}

	return peers.Unmarshal([]byte(trackerResp.Peers))
}

func (t *TorrentFile) DownloadToFile(path string) error {
	var peerID [20]byte
	_, err := rand.Read(peerID[:])
	if err != nil {
		return err
	}

	peers, err := t.requestPeers(peerID, Port)
	if err != nil {
		return err
	}

	torrent := p2p.Torrent{
		Peers:       peers,
		PeerID:      peerID,
		InfoHash:    t.InfoHash,
		PieceHashes: t.PieceHashes,
		PieceLength: t.PieceLength,
		Length:      t.Length,
		Name:        t.Name,
	}

	buff, err := torrent.Download()
	if err != nil {
		return err
	}

	outFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer outFile.Close()
	_, err = outFile.Write(buff)
	if err != nil {
		return err
	}
	return nil
}

func (i *bencodeInfo) hash() ([20]byte, error) {
	var buf bytes.Buffer
	err := bencode.Marshal(&buf, i)
	if err != nil {
		return [20]byte{}, err
	}

	h := sha1.Sum(buf.Bytes())
	return h, nil
}

func Open(path string) (TorrentFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return TorrentFile{}, err
	}
	defer file.Close()
	bto := bencodeTorrent{}
	err = bencode.Unmarshal(file, &bto)
	if err != nil {
		return TorrentFile{}, err
	}
	return bto.ToTorrentFile()
}

func (i *bencodeInfo) splitPiecesHashes() ([][20]byte, error) {
	hashLen := 20
	buf := []byte(i.Pieces)
	if len(buf)%hashLen != 0 {
		return nil, fmt.Errorf("received malformed pieces hashes")
	}

	numHashes := len(buf) / hashLen
	hashes := make([][20]byte, numHashes)
	for i := 0; i < numHashes; i++ {
		copy(hashes[i][:], buf[i*hashLen:(i+1)*hashLen])
	}

	return hashes, nil
}

func (bto bencodeTorrent) ToTorrentFile() (TorrentFile, error) {
	// Convert the pieces string into a slice of 20-byte hashes
	infoHash, err := bto.Info.hash()
	if err != nil {
		return TorrentFile{}, err
	}
	pieceHashes, err := bto.Info.splitPiecesHashes()
	if err != nil {
		return TorrentFile{}, err
	}

	t := TorrentFile{
		Announce:    bto.Announce,
		InfoHash:    infoHash,
		PieceHashes: pieceHashes,
		PieceLength: bto.Info.PieceLength,
		Length:      bto.Info.Length,
		Name:        bto.Info.Name,
	}

	return t, nil
}

// func Open(r io.Reader) (*bencodeTorrent, error) {
// 	bto := bencodeTorrent{}
// 	err := bencode.Unmarshal(r, bto)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	return &bto, err
// }
//
