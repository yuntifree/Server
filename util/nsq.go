package util

import (
	"log"

	simplejson "github.com/bitly/go-simplejson"
	nsq "github.com/nsqio/go-nsq"
)

const (
	requestType  = 1
	responseType = 2
)

//NewNsqProducer return nsq producer
func NewNsqProducer() *nsq.Producer {
	config := nsq.NewConfig()
	w, err := nsq.NewProducer("10.26.210.175:4150", config)
	if err != nil {
		log.Fatal(err)
	}
	err = w.Ping()
	if err != nil {
		log.Fatal(err)
	}
	return w
}

func packData(api string, ptype, code int64) ([]byte, error) {
	js, err := simplejson.NewJson([]byte(`{}`))
	if err != nil {
		log.Printf("packData NewJson failed:%v", err)
		return []byte(""), err
	}
	js.Set("name", api)
	js.Set("type", ptype)
	host := GetInnerIP()
	js.Set("host", host)
	if ptype == responseType {
		js.Set("code", code)
	}
	data, err := js.Encode()
	if err != nil {
		log.Printf("packData Encode failed:%v", err)
		return []byte(""), err
	}
	return data, nil
}

//PubRequest publish request to nsq
func PubRequest(w *nsq.Producer, api string) error {
	data, err := packData(api, requestType, 0)
	if err != nil {
		return err
	}
	err = w.Publish("api_monitor", data)
	if err != nil {
		log.Printf("PubRequest Publish failed:%v", err)
		return err
	}
	return nil
}

//PubResponse publish request to nsq
func PubResponse(w *nsq.Producer, api string, code int64) error {
	data, err := packData(api, responseType, code)
	if err != nil {
		return err
	}
	err = w.Publish("api_monitor", data)
	if err != nil {
		log.Printf("PubResponse Publish failed:%v", err)
		return err
	}
	return nil
}
