package main

import (
	"net/http"

	"../httpserver"

	"github.com/facebookgo/grace/gracehttp"
)

func main() {
	gracehttp.Serve(
		&http.Server{Addr: ":80", Handler: httpserver.NewAppServer()},
	)
}
