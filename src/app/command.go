package app

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

func RunCLI() {
	var self Client
	var init Client
	init.Init(20000)
	init.node.Create()
	var port int
	var f *os.File
	f, _ = os.Create("/tmp/test.log")
	logrus.SetOutput(f)
	blue.Println("Welcome this poor bitTorrent!")

	blue.Printf("Please Input your port number: ")
	fmt.Scanln(&port)
	self.Init(port)

	blue.Println("Type \"help\" for more info")
	for {
		var (
			op       = ""
			element1 = ""
			element2 = ""
			element3 = ""
		)
		fmt.Scanln(&op, &element1, &element2, &element3)
		if op == "create" {
			self.node.Create()
			green.Println("Create network succeed >w<")
			continue
		}
		if op == "join" {
			ok := self.node.Join(element1)
			if ok {
				green.Println("Join succeed >v<")
			} else {
				red.Println("Join failed >x<")
			}
			continue
		}
		if op == "upload" {
			self.Upload(element1, element2)
			green.Println("Upload succeed >o<")
			continue
		}
		if op == "download" {
			if element1 == "-t" {
				self.DownLoadByTorrent(element2, element3)
			} else if element1 == "-m" {
				self.DownLoadByMagnet(element2, element3)
			}
			green.Println("Download succeed >u<")
			continue
		}
		if op == "quit" {
			self.node.Quit()
			green.Println("Quit succeed >^<")
			continue
		}
		if op == "exit" {
			green.Println("Bye~ :)")
			goto exit
		}
		if op == "help" {
			yellow.Println("<-------------------------------------------------------------------------------->")
			yellow.Println("  join        [address]                     # Join the network                    ")
			yellow.Println("  create                                    # Create a new network                ")
			yellow.Println("  upload      [file path]   [torrent path]  # Upload file and generate .torrent   ")
			yellow.Println("  download -t [torrent path]   [file path]  # Download file by .torrent           ")
			yellow.Println("  download -m [magnet string]  [file path]  # Download file by magnet             ")
			yellow.Println("  quit                                      # Quit the network                    ")
			yellow.Println("  exit                                      # Exit the program                    ")
			yellow.Println("<-------------------------------------------------------------------------------->")
			continue
		}
	}
exit:
}
