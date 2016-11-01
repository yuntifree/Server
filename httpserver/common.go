package httpserver

import (
	"log"
	"net/http"

	util "../util"
	simplejson "github.com/bitly/go-simplejson"
)

const (
	helloAddress    = "localhost:50051"
	verifyAddress   = "localhost:50052"
	hotAddress      = "localhost:50053"
	discoverAddress = "localhost:50054"
	fetchAddress    = "localhost:50055"
	defaultName     = "world"
)

type appHandler func(http.ResponseWriter, *http.Request) *util.AppError

func (fn appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if e := fn(w, r); e != nil {
		log.Printf("error type:%d code:%d msg:%s", e.Type, e.Code, e.Msg)

		js, _ := simplejson.NewJson([]byte(`{}`))
		js.Set("errcode", e.Code)
		js.Set("desc", e.Msg)
		body, err := js.MarshalJSON()
		if err != nil {
			log.Printf("MarshalJSON failed: %v", err)
			w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
			return
		}
		w.Write(body)
	}
}
