package weixin

import (
	"Server/util"
	"fmt"
	"log"

	simplejson "github.com/bitly/go-simplejson"
)

const (
	qrURL = "https://api.weixin.qq.com/cgi-bin/wxaapp/createwxaqrcode"
)

//CreateQRCode create qrcode
func CreateQRCode(accesstoken, path string, width int64) (string, error) {
	url := fmt.Sprintf("%s?access_token=%s", qrURL, accesstoken)
	js, err := simplejson.NewJson([]byte(`{}`))
	if err != nil {
		log.Printf("CreateQRCode NewJson failed:%v", err)
		return "", err
	}
	js.Set("path", path)
	js.Set("width", width)
	body, err := js.Encode()
	if err != nil {
		log.Printf("CreateQRCode Encode body failed:%v", err)
		return "", err
	}
	resp, err := util.HTTPRequest(url, string(body))
	if err != nil {
		log.Printf("CreateQRCode HTTPRequest failed:%v", err)
		return "", err
	}
	return resp, nil
}
