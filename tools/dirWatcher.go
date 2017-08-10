package main

import (
	"Server/aliyun"
	"Server/util"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/rjeczalik/notify"
)

func uploadFile(path string) bool {
	filename := util.ExtractFilename(path)
	return aliyun.UploadOssImgFromFile(filename, path)
}

func main() {
	dir := flag.String("dir", "/data/dev/html", "image directory")
	flag.Parse()
	c := make(chan notify.EventInfo, 100)
	if err := notify.Watch(*dir, c, notify.InCloseWrite); err != nil {
		log.Fatal(err)
	}
	defer notify.Stop(c)
	for {
		select {
		case ei := <-c:
			log.Printf("Got event:%+v", ei)
			path := ei.Path()
			fi, err := os.Stat(path)
			if err != nil {
				log.Printf("stat failed:%s %v", path, err)
				continue
			}
			switch mode := fi.Mode(); {
			case mode.IsRegular():
				fmt.Println("regualar file:", path)
				if !uploadFile(path) {
					fmt.Println("uploadFile failed:", path)
				}
			case mode.IsDir():
				fmt.Println("directory:", path)
			case mode&os.ModeSymlink != 0:
				fmt.Println("symbolic link:", path)
			case mode&os.ModeNamedPipe != 0:
				fmt.Println("named pipe:", path)
			}
		}
	}
}
