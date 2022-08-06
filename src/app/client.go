package app

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/jackpal/bencode-go"
)

type Client struct {
	node dhtNode
}

func (n *Client) Init(port int) {
	n.node = NewNode(port)
	n.node.Run()
	yellow.Printf("Client start running at %s:%d\n", GetLocalAddress(), port)
}

func (n *Client) DownLoadByMagnet(magnet string, savePath string) {
	ok, torrentString := n.node.Get(magnet)
	if ok {
		reader := bytes.NewBufferString(torrentString)
		torrent := bencodeTorrent{}
		err := bencode.Unmarshal(reader, &torrent)
		if err != nil {
			red.Printf("Unmarshal torrent error: %v\n", err)
		} else {
			torrent.Save("tmp.torrent")
			green.Println("Save torrent file to tmp.torrent successfully")
		}
	} else {
		red.Println("Failed to find torrent of given magnet URL")
	}
	n.DownLoadByTorrent("tmp.torrent", savePath)
}

func (n *Client) DownLoadByTorrent(torrentPath string, savePath string) {
	file, err := os.Create(savePath)
	if err != nil {
		red.Printf("Create file %s error: %v\n", savePath, err)
		return
	}
	torrent, err := Open(torrentPath)
	if err != nil {
		red.Printf("Failed to open torrent in path %s, error: %v\n", torrentPath, err.Error())
		return
	}
	key := torrent.makeKeyPackage()
	green.Printf("File name: %s, size: %v bytes, piece num: %v\n", torrent.Info.Name, torrent.Info.Length, key.size)

	ok, data := n.DownLoad(&key)
	if ok {
		file.Write(data)
		green.Printf("Download %s to %s successfully\n", torrent.Info.Name, savePath)
	} else {
		red.Println("Download failed.")
	}
}

func (n *Client) Upload(filePath, torrentPath string) {
	yellow.Printf("Start uploading file %s\n", filePath)
	dataPackage, length := makeDataPackage(filePath)
	cyan.Println("Data packaged, piece number: ", dataPackage.size)

	var pieces string
	for i := 0; i < dataPackage.size; i++ {
		piece, _ := PiecesHash(dataPackage.data[i], i)
		pieces += fmt.Sprintf("%x", piece)
	}
	_, fileName := path.Split(filePath)
	torrent := bencodeTorrent{
		Announce: "",
		Info: bencodeInfo{
			Length:       length,
			Pieces:       pieces,
			PiecesLength: PieceSize,
			Name:         fileName,
		},
	}

	err := torrent.Save(torrentPath)
	if err != nil {
		red.Printf("Save torrent file error: %v\n", err)
		return
	}

	key := torrent.makeKeyPackage()
	green.Printf("File name: %s, size: %v bytes, piece num: %v\n", torrent.Info.Name, torrent.Info.Length, key.size)

	var buf bytes.Buffer
	err = bencode.Marshal(&buf, torrent)
	if err != nil {
		red.Printf("Torrent marshal error: %v\n", err)
	}
	magnet := MakeMagnet(fmt.Sprintf("%x", key.infoHash))
	torrentString := buf.String()

	ok := n.UploadPackage(&key, &dataPackage)
	if ok {
		green.Printf("Upload finished, create torrent file: %s\n", torrentPath)
		n.node.Put(magnet, torrentString)
		cyan.Printf("Magnet URL: %s\n", magnet)
	}
}

type pieceWork struct {
	index   int
	retry   int
	success bool
	result  DataPiece
}

func (n *Client) UploadPieceWork(keyPackage *KeyPackage, dataPackage *DataPackage, index int, retry int, ch *chan pieceWork) {
	key := keyPackage.getKey(index)
	ok := n.node.Put(key, string(dataPackage.data[index]))
	if ok {
		*ch <- pieceWork{index: index, success: true, retry: retry}
	} else {
		*ch <- pieceWork{index: index, success: false, retry: retry}
	}
}

func (n *Client) UploadPackage(keyPackage *KeyPackage, dataPackage *DataPackage) bool {
	ch := make(chan pieceWork, QueueLength)
	for i := 0; i < keyPackage.size; i++ {
		go n.UploadPieceWork(keyPackage, dataPackage, i, 0, &ch)
	}
	donePieces := 0
	for donePieces < keyPackage.size {
		select {
		case piece := <-ch:
			if piece.success {
				donePieces++
				green.Printf("Piece #%d uploaded successfully\n", piece.index+1)
			} else {
				red.Printf("Piece #%d upload failed %d/%d\n", piece.index+1, piece.retry, UploadRetryNum)
				if piece.retry < UploadRetryNum {
					go n.UploadPieceWork(keyPackage, dataPackage, piece.index, piece.retry+1, &ch)
				} else {
					red.Println("Upload failed :(")
					return false
				}
			}
		case <-time.After(UploadTimeout * time.Duration(keyPackage.size)):
			red.Println("Upload timeout :(")
			return false
		}
	}
	green.Println("Upload finished :)")
	return true
}

func (c *Client) DownloadPieceWork(k *KeyPackage, index int, retry int, workQueue *chan pieceWork) {
	key := k.getKey(index)
	ok, piece := c.node.Get(key)
	if ok {
		*workQueue <- pieceWork{index: index, success: true, result: []byte(piece), retry: retry}
	} else {
		*workQueue <- pieceWork{index: index, success: false, retry: retry}
	}
}

func (n *Client) DownLoad(keyPackage *KeyPackage) (bool, []byte) {
	ret := make([]byte, keyPackage.length)
	ch := make(chan pieceWork, QueueLength)
	for i := 0; i < keyPackage.size; i++ {
		go n.DownloadPieceWork(keyPackage, i, 0, &ch)
	}

	donePieces := 0
	for donePieces < keyPackage.size {
		select {
		case piece := <-ch:
			if piece.success {
				donePieces++
				bound := (piece.index + 1) * PieceSize
				if bound > keyPackage.length {
					bound = keyPackage.length
				}
				copy(ret[piece.index*PieceSize:bound], piece.result)
				pieceHash, _ := PiecesHash(piece.result, piece.index)
				if pieceHash != keyPackage.key[piece.index] {
					red.Printf("Check integrity of Piece #%d failed\n", piece.index+1)
					return false, []byte{}
				}
				green.Printf("Piece #%d downloaded successfully\n", piece.index+1)
			} else {
				red.Printf("Piece #%d download failed %d/%d\n", piece.index+1, piece.retry, DownloadRetryNum)
				if piece.retry < DownloadRetryNum {
					go n.DownloadPieceWork(keyPackage, piece.index, piece.retry+1, &ch)
				} else {
					red.Println("Download failed :(")
					return false, []byte{}
				}
			}
		case <-time.After(DownloadTimeout * time.Duration(keyPackage.size)):
			red.Println("Download timeout :(")
			return false, []byte{}

		}
	}

	green.Println("File Downloaded :)")
	return true, ret
}
