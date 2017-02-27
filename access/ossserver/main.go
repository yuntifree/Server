package main

import (
	"net/http"

	"github.com/facebookgo/grace/gracehttp"
)

func main() {
	gracehttp.Serve(
		&http.Server{Addr: ":8080", Handler: NewOssServer()},
	)
}
