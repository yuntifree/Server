package main

import (
	"Server/aliyun"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/rjeczalik/notify"
)

func extractFilename(path string) string {
	pos := strings.LastIndex(path, "/")
	if pos != -1 {
		return path[pos+1:]
	}
	return path
}

func uploadFile(path string) bool {
	filename := extractFilename(path)
	return aliyun.UploadOssImgFromFile(filename, path)
}

func main() {
	dir := flag.String("dir", "/data/dev/html", "image directory")
	flag.Parse()
	c := make(chan notify.EventInfo, 100)
	if err := notify.Watch(*dir, c, notify.Create); err != nil {
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
