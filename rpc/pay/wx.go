package main

import (
	"Server/proto/common"
	"Server/proto/pay"
	"Server/util"
	"Server/weixin"
	"database/sql"
	"log"
	"time"

	"golang.org/x/net/context"
)

func recordOrderInfo(db *sql.DB, oid string, in *pay.WxPayRequest) (int64, error) {
	res, err := db.Exec("INSERT INTO orders(oid, uid, tuid, type, item, price, ctime) VALUES (?, ?, ?, ?, ?, ?, NOW())",
		oid, in.Head.Uid, in.Tuid, in.Type, in.Item, in.Fee)
	if err != nil {
		log.Printf("recordOrderInfo failed:%s %v", oid, err)
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Printf("recordOrderInfo get insert id failed:%s %v", oid, err)
		return 0, err
	}
	return id, nil
}

func (s *server) WxPay(ctx context.Context, in *pay.WxPayRequest) (*pay.WxPayReply, error) {
	log.Printf("WxPay request:%+v", in)
	util.PubRPCRequest(w, "pay", "WxPay")
	oid := weixin.GenOrderID(in.Head.Uid)
	log.Printf("WxPay request:%+v oid:%s", in, oid)
	_, err := recordOrderInfo(db, oid, in)
	if err != nil {
		return &pay.WxPayReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil

	}

	var req weixin.UnifyOrderReq
	req.Appid = weixin.InquiryAppid
	req.Body = "问诊打赏"
	req.MchID = weixin.InquiryMerID
	req.NonceStr = util.GenSalt()
	req.Openid = in.Openid
	req.TradeType = "JSAPI"
	req.SpbillCreateIP = in.Clientip
	req.TotalFee = in.Fee
	req.OutTradeNO = oid

	resp, err := weixin.UnifyPayRequest(req)
	if err != nil {
		log.Printf("WxPay UnifyPayRequest failed:%v", err)
		return &pay.WxPayReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	log.Printf("resp:%+v", resp)
	if resp.ReturnCode != "SUCCESS" || resp.ResultCode != "SUCCESS" {
		log.Printf("WxPay UnifyPayRequest failed message:%s", resp.ReturnMsg)
		return &pay.WxPayReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}

	now := time.Now().Unix()
	m := make(map[string]interface{})
	m["appId"] = resp.Appid
	m["nonceStr"] = resp.NonceStr
	m["package"] = "prepay_id=" + resp.PrepayID
	m["signType"] = "MD5"
	m["timeStamp"] = now
	sign := weixin.CalcSign(m, weixin.InquiryMerKey)

	util.PubRPCSuccRsp(w, "pay", "WxPay")
	return &pay.WxPayReply{
		Head:    &common.Head{Retcode: 0, Uid: in.Head.Uid},
		Package: "pre_payid=" + resp.PrepayID, NonceStr: resp.NonceStr,
		TimeStamp: now, PaySign: sign, SignType: "MD5",
	}, nil
}

func (s *server) WxPayCB(ctx context.Context, in *pay.WxPayCBRequest) (*common.CommReply, error) {
	log.Printf("WxPayCB request:%+v", in)
	util.PubRPCRequest(w, "pay", "WxPayCB")
	var ptype, pid, status int64
	err := db.QueryRow("SELECT type, item, status FROM orders WHERE oid = ?", in.Oid).
		Scan(&ptype, &pid, &status)
	if err != nil {
		log.Printf("WxPayCB query order info failed:%v", err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1},
		}, nil
	}
	if status == 1 {
		log.Printf("WxPayCB has duplicate oid:%s", in.Oid)
		return &common.CommReply{
			Head: &common.Head{Retcode: 0},
		}, nil
	}
	_, err = db.Exec("UPDATE orders SET status = 1, fee = ?, ftime = NOW() WHERE id = ?", in.Fee, pid)
	if err != nil {
		log.Printf("WxPayCB update order status failed::%s %v", in.Oid, err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1},
		}, nil
	}
	_, err = db.Exec("UPDATE inquiry_history SET status = 1 WHERE id = ?", pid)
	if err != nil {
		log.Printf("WxPayCB update inquiry history failed:%d %v", pid, err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1},
		}, nil
	}
	util.PubRPCSuccRsp(w, "pay", "WxPayCB")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0},
	}, nil
}
