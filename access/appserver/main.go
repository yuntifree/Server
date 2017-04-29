package main

import (
	"Server/util"
	"crypto/tls"
	"log"
	"net/http"
	"time"

	"github.com/facebookgo/grace/gracehttp"
)

const (
	certPath    = "/data/server/fullchain.pem"
	privKeyPath = "/data/server/privkey.pem"
)

func init() {
	w := util.NewRotateWriter("/data/server/app.log", 1024*1024*1024)
	log.SetOutput(w)
}

func main() {
	cer, err := tls.LoadX509KeyPair(certPath, privKeyPath)
	if err != nil {
		log.Println(err)
		return
	}
	config := &tls.Config{Certificates: []tls.Certificate{cer}}
	gracehttp.Serve(
		&http.Server{Addr: ":80", Handler: NewAppServer(), IdleTimeout: 30 * time.Second},
		&http.Server{Addr: ":443", Handler: NewAppServer(), TLSConfig: config, IdleTimeout: 30 * time.Second},
	)
}
