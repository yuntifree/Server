package weixin

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"sort"
	"strings"
)

const (
	orderURL      = "https://api.mch.weixin.qq.com/pay/unifiedorder"
	packValue     = "Sign=WXPay"
	InquiryMerID  = "1482126772"
	InquiryMerKey = "AB1640D05DD44FCBB448EBBEE03274E3"
	callbackURL   = "https://api.yunxingzh.com/inquiry/wx_pay_callback"
)

//UnifyOrderReq unify order request
type UnifyOrderReq struct {
	Appid          string `xml:"appid"`
	Body           string `xml:"body"`
	MchID          string `xml:"mch_id"`
	NonceStr       string `xml:"nonce_str"`
	NotifyURL      string `xml:"notify_url"`
	TradeType      string `xml:"trade_type"`
	SpbillCreateIP string `xml:"spbill_create_ip"`
	TotalFee       int64  `xml:"total_fee"`
	OutTradeNO     string `xml:"out_trade_no"`
	Sign           string `xml:"sign"`
	Openid         string `xml:"openid"`
}

//UnifyOrderResp unify order response
type UnifyOrderResp struct {
	ReturnCode string `xml:"return_code"`
	ReturnMsg  string `xml:"return_msg"`
	Appid      string `xml:"appid"`
	MchID      string `xml:"mch_id"`
	NonceStr   string `xml:"nonce_str"`
	Openid     string `xml:"openid"`
	Sign       string `xml:"sign"`
	ResultCode string `xml:"result_code"`
	TradeType  string `xml:"trade_type"`
	PrepayID   string `xml:"prepay_id"`
}

//NotifyRequest notify request
type NotifyRequest struct {
	ReturnCode    string `xml:"return_code"`
	ReturnMsg     string `xml:"return_msg"`
	Appid         string `xml:"appid"`
	MchID         string `xml:"mch_id"`
	NonceStr      string `xml:"nonce_str"`
	Openid        string `xml:"openid"`
	Sign          string `xml:"sign"`
	ResultCode    string `xml:"result_code"`
	TradeType     string `xml:"trade_type"`
	BankType      string `xml:"bank_type"`
	TotalFee      int64  `xml:"total_fee"`
	CashFee       int64  `xml:"cash_fee"`
	TranscationID string `xml:"transaction_id"`
	OutTradeNO    string `xml:"out_trade_no"`
	TimeEnd       string `xml:"time_end"`
}

//VerifyNotify verify notify sign
func VerifyNotify(req NotifyRequest) bool {
	vt := reflect.TypeOf(req)
	vv := reflect.ValueOf(req)
	m := make(map[string]interface{})

	for i := 0; i < vt.NumField(); i++ {
		f := vt.Field(i)
		name := f.Tag.Get("xml")
		m[name] = vv.FieldByName(f.Name).String()
	}
	sign := CalcSign(m, InquiryMerKey)
	if req.Sign != sign {
		return false
	}
	return true
}

//CalcSign calc md5 sign
func CalcSign(mReq map[string]interface{}, key string) string {
	var sortedKeys []string
	for k := range mReq {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	var signStr string
	for _, k := range sortedKeys {
		log.Printf("%v -- %v", k, mReq[k])
		value := fmt.Sprintf("%v", mReq[k])
		if value != "" {
			signStr += k + "=" + value + "&"
		}
	}

	if key != "" {
		signStr += "key=" + key
	}

	md5Ctx := md5.New()
	md5Ctx.Write([]byte(signStr))
	cipherStr := md5Ctx.Sum(nil)
	upperSign := strings.ToUpper(hex.EncodeToString(cipherStr))
	return upperSign
}

func calcReqSign(req UnifyOrderReq, merKey string) string {
	m := make(map[string]interface{})
	m["appid"] = req.Appid
	m["body"] = req.Body
	m["mch_id"] = req.MchID
	m["notify_url"] = req.NotifyURL
	m["trade_type"] = req.TradeType
	m["spbill_create_ip"] = req.SpbillCreateIP
	m["total_fee"] = req.TotalFee
	m["out_trade_no"] = req.OutTradeNO
	m["nonce_str"] = req.NonceStr
	return CalcSign(m, merKey)
}

//UnifyPayRequest send unify order pay request
func UnifyPayRequest(req UnifyOrderReq) (*UnifyOrderResp, error) {
	req.Sign = calcReqSign(req, InquiryMerKey)

	buf, err := xml.Marshal(req)
	if err != nil {
		log.Printf("UnifyPayRequest marshal failed:%v", err)
		return nil, err
	}

	reqStr := string(buf)
	reqStr = strings.Replace(reqStr, "XUnifyOrderReq", "xml", -1)

	request, err := http.NewRequest("POST", orderURL, bytes.NewReader([]byte(reqStr)))
	if err != nil {
		log.Printf("UnifyPayRequest NewRequest failed:%v", err)
		return nil, err
	}
	request.Header.Set("Accept", "application/xml")
	request.Header.Set("Content-Type", "application/xml;charset=utf-8")

	c := http.Client{}
	resp, err := c.Do(request)
	if err != nil {
		log.Printf("UnifyPayRequest request failed:%v", err)
		return nil, err
	}

	defer resp.Body.Close()
	dec := xml.NewDecoder(resp.Body)
	var res UnifyOrderResp
	err = dec.Decode(&res)
	if err != nil {
		log.Printf("UnifyPayRequest Unmarshal failed:%v", err)
		return nil, err
	}
	return &res, nil
}
