package util

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	simplejson "github.com/bitly/go-simplejson"
)

const (
	tplID       = 1820708
	apiKey      = "87fda7cd28aad018688c3ce04bbf1df2"
	tplURL      = "https://sms.yunpian.com/v2/sms/tpl_single_send.json"
	healthTplID = 1840358
	payTplID    = 1844680
)

//SendReserveSMS send reserve info sms
func SendReserveSMS(mobile, verifycode, stime string) error {
	return SendYPSMS(mobile, verifycode, stime, tplID)
}

//SendPaySMS send payed notify sms
func SendPaySMS(mobile string) error {
	data := url.Values{"apikey": {apiKey}, "mobile": {mobile},
		"tpl_id": {fmt.Sprintf("%d", payTplID)}}
	return sendData(data)
}

//SendHealthSMS send health info sms
func SendHealthSMS(mobile, verifycode string) error {
	tplValue := url.Values{"#code#": {verifycode}}.Encode()
	data := url.Values{"apikey": {apiKey}, "mobile": {mobile},
		"tpl_id": {fmt.Sprintf("%d", healthTplID)}, "tpl_value": {tplValue}}
	return sendData(data)
}

//SendYPSMS use yunpian to send sms
func SendYPSMS(mobile, verifycode, stime string, tmpID int64) error {
	tplValue := url.Values{"#code#": {verifycode}, "#time#": {stime}}.Encode()
	data := url.Values{"apikey": {apiKey}, "mobile": {mobile},
		"tpl_id": {fmt.Sprintf("%d", tmpID)}, "tpl_value": {tplValue}}
	return sendData(data)
}

func sendData(data url.Values) error {
	resp, err := http.PostForm(tplURL, data)
	if err != nil {
		log.Printf("SendYPSMS request failed:%v", err)
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("SendYPSMS read response failed:%v", err)
		return err
	}
	js, err := simplejson.NewJson(body)
	if err != nil {
		log.Printf("SendYPSMS parse response failed:%v", err)
		return err
	}
	code, err := js.Get("code").Int()
	if err != nil {
		log.Printf("SendYPSMS get response code failed:%v", err)
		return err
	}
	if code != 0 {
		log.Printf("SendYPSMS illegal code:%s", string(body))
		return fmt.Errorf("illegal response code:%d", code)
	}
	return nil
}
