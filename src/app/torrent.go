package app

import (
	"bytes"
	"crypto/sha1"
	"os"
	"strconv"

	"github.com/jackpal/bencode-go"
)

//from: https://blog.mynameisdhr.com/YongGOCongLingJianLiBitTorrentKeHuDuan/

type bencodeInfo struct {
	Pieces       string `bencode:"pieces"`
	PiecesLength int    `bencode:"piece length"`
	Length       int    `bencode:"Length"`
	Name         string `bencode:"name"`
}

type bencodeTorrent struct {
	Announce string      `bencode:"announce"`
	Info     bencodeInfo `bencode:"info"`
}

func Open(path string) (*bencodeTorrent, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	bto := &bencodeTorrent{}
	err = bencode.Unmarshal(file, bto)
	if err != nil {
		return nil, err
	}
	return bto, nil
}

func (i *bencodeInfo) InfoHash() ([SHA1Len]byte, error) {
	var buf bytes.Buffer
	err := bencode.Marshal(&buf, *i)
	if err != nil {
		return [20]byte{}, err
	}
	h := sha1.Sum(buf.Bytes())
	return h, nil
}

func PiecesHash(piece DataPiece, index int) ([SHA1Len]byte, error) {
	piece = append(piece, []byte(strconv.Itoa(index))...)
	pieceHash := sha1.Sum(piece)
	return pieceHash, nil
}

func (t *bencodeTorrent) Save(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	torrent := bencodeTorrent{
		Announce: "",
		Info:     t.Info,
	}
	err = bencode.Marshal(f, torrent)
	if err != nil {
		return err
	}
	return nil
}
