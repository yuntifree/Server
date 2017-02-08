package main

import (
	"crypto/tls"
	"log"
	"net/http"

	"Server/httpserver"

	"github.com/facebookgo/grace/gracehttp"
)

const (
	certPath    = "/data/server/fullchain.pem"
	privKeyPath = "/data/server/privkey.pem"
)

func main() {
	cer, err := tls.LoadX509KeyPair(certPath, privKeyPath)
	if err != nil {
		log.Println(err)
		return
	}
	config := &tls.Config{Certificates: []tls.Certificate{cer}}
	gracehttp.Serve(
		&http.Server{Addr: ":80", Handler: httpserver.NewAppServer()},
		&http.Server{Addr: ":443", Handler: httpserver.NewAppServer(), TLSConfig: config},
	)
}
