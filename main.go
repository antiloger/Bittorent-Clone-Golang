package main

import (
	"log"
	"os"

	"github.com/antiloger/bittorrent-clone-golang/torrent"
)

func main() {
	inpath := os.Args[1]
	outpath := os.Args[2]

	tf, err := torrent.Open(inpath)
	if err != nil {
		log.Fatal(err)
	}

	err = tf.DownloadToFile(outpath)
	if err != nil {
		log.Fatal(err)
	}
}
