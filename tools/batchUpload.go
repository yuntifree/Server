package main

import (
	"Server/aliyun"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
