package main

import (
	"net/http"

	"Server/httpserver"

	"github.com/facebookgo/grace/gracehttp"
)

func main() {
	gracehttp.Serve(
		&http.Server{Addr: ":8080", Handler: httpserver.NewOssServer()},
	)
}
