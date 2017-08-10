package main

import (
	"Server/util"
	"encoding/json"
	"flag"
	"log"
	"os"
	"strings"
)

const (
	getURL = "http://120.76.236.185/get_online_loginimg"
	ackURL = "http://120.76.236.185/ack_loginimg"
	uid    = 137
	token  = "6ba9ac5a422d4473b337d57376dd3488"
	defDir = "/data/tmp/"
)

type GetRequest struct {
	Uid   int64  `json:"uid"`
	Token string `json:"token"`
}

type GetResponse struct {
	Errno int64    `json:"errno"`
	Data  GetReply `json:"data"`
}

type GetReply struct {
	Infos []Image `json:"infos"`
}

type Image struct {
	Id  int64  `json:"id"`
	Img string `json:"img"`
}

type AckRequest struct {
	Uid   int64  `json:"uid"`
	Token string `json:"token"`
	Id    int64  `json:"id"`
}

type AckResponse struct {
	Errno int64  `json:"errno"`
	Desc  string `json:"desc"`
}

func main() {
	dir := flag.String("dir", defDir, "image directory")
	flag.Parse()
	images := getOnlineImages()
	log.Printf("images:%+v", images)
	for _, v := range images {
		handleImage(v, *dir)
	}
}

func getOnlineImages() []Image {
	var req GetRequest
	req.Uid = uid
	req.Token = token
	body, err := json.Marshal(req)
	if err != nil {
		log.Printf("json marshal failed:%v", err)
		return nil
	}

	resp, err := util.HTTPRequest(getURL, string(body))
	if err != nil {
		log.Printf("HTTPRequest failed:%v", err)
		return nil
	}

	var rsp GetResponse
	err = json.Unmarshal([]byte(resp), &rsp)
	if err != nil {
		log.Printf("json unmarshal failed:%s %v", resp, err)
		return nil
	}
	return rsp.Data.Infos
}

func extractFilename(path string) string {
	pos := strings.LastIndex(path, "/")
	if pos != -1 {
		return path[pos+1:]
	}
	return path
}

func existsFile(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

func ackLoginImg(id int64) {
	var req AckRequest
	req.Uid = uid
	req.Token = token
	req.Id = id
	body, err := json.Marshal(req)
	if err != nil {
		log.Printf("json marshal failed:%v", err)
		return
	}

	resp, err := util.HTTPRequest(ackURL, string(body))
	if err != nil {
		log.Printf("HTTPRequest failed:%v", err)
		return
	}

	var rsp AckResponse
	err = json.Unmarshal([]byte(resp), &rsp)
	if err != nil {
		log.Printf("json unmarshal failed:%v", err)
		return
	}
	if rsp.Errno != 0 {
		log.Printf("failure response errno:%d", rsp.Errno)
	}
}

func handleImage(img Image, dir string) {
	filename := extractFilename(img.Img)
	path := dir + filename
	if existsFile(path) {
		log.Printf("path:%s exists", filename)
		ackLoginImg(img.Id)
		return
	}
	resp, err := util.HTTPRequest(img.Img, "")
	if err != nil {
		log.Printf("get image:%s failed:%v", img.Img, err)
		return
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Printf("open file:%s failed:%v", path, err)
		return
	}
	defer f.Close()
	_, err = f.WriteString(resp)
	if err != nil {
		log.Printf("write string failed:%s %v", path, err)
		return
	}

	ackLoginImg(img.Id)
}
