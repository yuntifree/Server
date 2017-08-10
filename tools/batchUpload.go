package main

import (
	"Server/aliyun"
	"Server/util"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func uploadFile(path string) bool {
	filename := util.ExtractFilename(path)
	return aliyun.UploadOssImgFromFile(filename, path)
}

func main() {
	dir := flag.String("dir", "/data/tmp", "upload directory")
	flag.Parse()
	filepath.Walk(*dir, func(path string, info os.FileInfo, err error) error {
		mode := info.Mode()
		if mode.IsRegular() {
			fmt.Println("regualar file:", path)
			if !uploadFile(path) {
				fmt.Println("uploadFile failed:", path)
			}
		}
		return nil
	})
}
