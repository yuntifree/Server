package main

import (
	"Server/aliyun"
	"Server/util"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	dir     = "/data/seaportsp/attachment/images"
	baseURL = "http://yuntiimgs.oss-cn-shenzhen-internal.aliyuncs.com"
)

func main() {
	now := time.Now()
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		mode := info.Mode()
		mtime := info.ModTime()
		if mode.IsRegular() && !mtime.Before(now.Add(-5*60*time.Second)) {
			fmt.Println("regualar file:", path)
			filename := util.ExtractFilename(path)
			if !hasUploadFile(filename) {
				fmt.Println("upload file:", path)
				if !uploadFile(path) {
					fmt.Println("uploadFile failed:", path)
				}
			}
		}
		return nil
	})
}

func uploadFile(path string) bool {
	filename := util.ExtractFilename(path)
	return aliyun.UploadOssImgFromFile(filename, path)
}

func hasUploadFile(filename string) bool {
	url := baseURL + "/" + filename
	client := &http.Client{}
	resp, err := client.Head(url)
	if err != nil {
		log.Printf("Head url:%s failed:%v", url, err)
		return false
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("Head url:%s status:%s", url, resp.Status)
		return false
	}
	return true
}
