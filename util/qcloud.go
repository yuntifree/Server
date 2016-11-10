package util

import (
	"fmt"
	"math/rand"
	"strconv"

	simplejson "github.com/bitly/go-simplejson"
)

const (
	appid  = "1400016615"
	appkey = "5acfd2d117cb98ed8cb30d9d8f7d32c3"
	smsurl = "https://yun.tim.qq.com/v3/tlssmssvr/sendsms"
)

func genBody(phone string, code int) string {
	js, err := simplejson.NewJson([]byte(`{"tel":{"nationcode":"86"}, "type":"0","ext":"","extend":""}`))
	if err != nil {
		return ""
	}
	s := fmt.Sprintf("%06d", code)
	msg := "【东莞无线】欢迎使用东莞无线免费WiFi,您的验证码为:" + s
	js.Set("msg", msg)
	sig := GetMD5Hash(appkey + phone)
	js.Set("sig", sig)
	js.SetPath([]string{"tel", "phone"}, phone)
	data, err := js.Encode()
	if err != nil {
		return ""
	}

	return string(data[:])
}

//SendSMS send verify code to phone
func SendSMS(phone string, code int) int {
	body := genBody(phone, code)
	fmt.Println(body)
	rand.Seed(42)
	url := smsurl + "?sdkappid=" + appid + "&random=" + strconv.Itoa(rand.Int())
	fmt.Println(url)
	rspbody, err := HTTPRequest(url, body)
	if err != nil {
		return -1
	}
	fmt.Println(string(rspbody))
	js, err := simplejson.NewJson([]byte(`{}`))
	err = js.UnmarshalJSON([]byte(rspbody))
	s, err := js.GetPath("result").String()
	if s != "0" {
		return -3
	}

	return 0
}
