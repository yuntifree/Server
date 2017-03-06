package main

import (
	"Server/util"
	"log"
	"os"

	simplejson "github.com/bitly/go-simplejson"
)

const (
	baseURL = "http://10.26.210.175/"
)

func main() {
	if len(os.Args) < 2 {
		log.Printf("not enough param")
		os.Exit(1)
	}
	f, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}
	defer f.Close()

	ofs, err := os.OpenFile("apiTest.log", os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		panic(err)
	}
	defer ofs.Close()

	js, err := simplejson.NewFromReader(f)
	if err != nil {
		panic(err)
	}

	arr := js.Get("api").MustArray()
	if err != nil {
		panic(err)
	}

	for i := 0; i < len(arr); i++ {
		name, err := js.Get("api").GetIndex(i).Get("name").String()
		req := js.Get("api").GetIndex(i).Get("req")
		log.Printf("name:%s", name)
		if err != nil {
			panic(err)
		}
		body, err := req.Encode()
		if err != nil {
			log.Printf("encode failed:%s %v", name, err)
			continue
		}
		url := baseURL + name
		resp, err := util.HTTPRequest(url, string(body))
		if err != nil {
			log.Printf("Test Failed:%s %v", name, err)
			continue
		}
		ofs.WriteString(name + "\n")
		ofs.WriteString(string(body) + "\n")
		ofs.WriteString(resp + "\n")
		rs, err := simplejson.NewJson([]byte(resp))
		if err != nil {
			log.Printf("parse response failed:%s %v", name, err)
			continue
		}
		errno, err := rs.Get("errno").Int64()
		if err != nil {
			log.Printf("get errno failed:%s %v", name, err)
			continue
		}
		if errno != 0 {
			log.Printf("API TEST FAILED:%s ERRNO:%d", name, errno)
		}
	}
	return
}
