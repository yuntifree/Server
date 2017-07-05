package main

import (
	"Server/util"
	"Server/weixin"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	simplejson "github.com/bitly/go-simplejson"
)

const (
	tmpURL   = "https://api.weixin.qq.com/cgi-bin/message/wxopen/template/send"
	payTmpID = "p4IzTAhg9hx6tlhDGSEHPIEmec0n_xcZt0_bf87Mi_4"
)

type msgResp struct {
	Errcode int64  `json:"errcode"`
	Errmsg  string `json:"errmsg"`
}

type keyVal struct {
	Value string `json:"value"`
	Color string `json:"color"`
}

type tmpData struct {
	Keyword1 keyVal `json:"keyword1"`
	Keyword2 keyVal `json:"keyword2"`
	Keyword3 keyVal `json:"keyword3"`
	Keyword4 keyVal `json:"keyword4"`
}

func sendPayWxMsg(db *sql.DB, openid, formID string, payInfos [4]string) {
	accessToken, err := weixin.RefreshAccessToken(db, weixin.InquiryAppid)
	if err != nil {
		log.Printf("sendPayWxMsg getAccessToken failed:%v", err)
		return
	}
	var data tmpData
	data.Keyword1.Value = payInfos[0]
	data.Keyword2.Value = payInfos[1]
	data.Keyword3.Value = payInfos[2]
	data.Keyword4.Value = payInfos[3]
	sendWxMsg(accessToken, openid, payTmpID, formID, data)
	return
}

func sendWxMsg(accessToken, openid, tmpID, formID string, data interface{}) {
	url := fmt.Sprintf("%s?access_token=%s", tmpURL, accessToken)
	js, err := simplejson.NewJson([]byte(`{}`))
	if err != nil {
		log.Printf("sendWxMsg NewJson failed:%v", err)
		return
	}
	js.Set("touser", openid)
	js.Set("template_id", tmpID)
	js.Set("form_id", formID)
	js.Set("data", data)
	body, err := js.Encode()
	if err != nil {
		log.Printf("sendWxMsg json encode failed:%v", err)
		return
	}
	resp, err := util.HTTPRequest(url, string(body))
	if err != nil {
		log.Printf("sendWxMsg HTTPRequest failed:%v", err)
		return
	}
	var res msgResp
	err = json.Unmarshal([]byte(resp), &res)
	if err != nil {
		log.Printf("sendWxMsg unmarshal response failed:%s %v", resp, err)
		return
	}
	if res.Errcode != 0 {
		log.Printf("sendWxMsg failed resp:%s", resp)
		return
	}
	return
}
