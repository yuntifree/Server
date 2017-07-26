package main

import (
	"Server/proto/common"
	"Server/proto/pay"
	"Server/util"
	"Server/weixin"
	"database/sql"
	"fmt"
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

func recordPrepayid(db *sql.DB, id int64, prepayid string) {
	_, err := db.Exec("UPDATE orders SET prepayid = ? WHERE id = ?",
		prepayid, id)
	if err != nil {
		log.Printf("recordPrepayid failed:%d %s %v", id, prepayid, err)
	}
}

func getUserOpenid(db *sql.DB, uid int64) (string, error) {
	var openid string
	err := db.QueryRow("SELECT w.openid FROM wx_openid w, users u WHERE u.username = w.unionid AND u.uid = ?", uid).Scan(&openid)
	if err != nil {
		log.Printf("getUserOpenid failed:%d %v", uid, err)
	}
	return openid, err
}

func getUserPhone(db *sql.DB, uid int64) (string, error) {
	var phone string
	err := db.QueryRow("SELECT phone FROM users WHERE uid = ?", uid).
		Scan(&phone)
	return phone, err
}

func (s *server) WxPay(ctx context.Context, in *pay.WxPayRequest) (*pay.WxPayReply, error) {
	log.Printf("WxPay request:%+v", in)
	util.PubRPCRequest(w, "pay", "WxPay")
	oid := weixin.GenOrderID(in.Head.Uid)
	log.Printf("WxPay request:%+v oid:%s", in, oid)
	id, err := recordOrderInfo(db, oid, in)
	if err != nil {
		return &pay.WxPayReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil

	}
	openid, err := getUserOpenid(db, in.Head.Uid)
	if err != nil {
		return &pay.WxPayReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil

	}

	var req weixin.UnifyOrderReq
	req.Appid = weixin.InquiryAppid
	req.Body = "咨询费"
	req.MchID = weixin.InquiryMerID
	req.NonceStr = util.GenSalt()
	req.Openid = openid
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

	recordPrepayid(db, id, resp.PrepayID)

	util.PubRPCSuccRsp(w, "pay", "WxPay")
	return &pay.WxPayReply{
		Head:    &common.Head{Retcode: 0, Uid: in.Head.Uid},
		Package: "prepay_id=" + resp.PrepayID, NonceStr: resp.NonceStr,
		TimeStamp: now, PaySign: sign, SignType: "MD5",
	}, nil
}

func (s *server) WxPayCB(ctx context.Context, in *pay.WxPayCBRequest) (*common.CommReply, error) {
	log.Printf("WxPayCB request:%+v", in)
	util.PubRPCRequest(w, "pay", "WxPayCB")
	var oid, ptype, pid, status, uid, tuid int64
	var prepayid string
	err := db.QueryRow("SELECT id,  type, item, status, uid, prepayid, tuid FROM orders WHERE oid = ?", in.Oid).
		Scan(&oid, &ptype, &pid, &status, &uid, &prepayid, &tuid)
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
	_, err = db.Exec("UPDATE orders SET status = 1, fee = ?, ftime = NOW() WHERE id = ?",
		in.Fee, oid)
	if err != nil {
		log.Printf("WxPayCB update order status failed::%s %v", in.Oid, err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1},
		}, nil
	}
	log.Printf("after update orders status:%s", in.Oid)
	_, err = db.Exec("UPDATE inquiry_history SET status = 1, ptime = NOW() WHERE id = ?", pid)
	if err != nil {
		log.Printf("WxPayCB update inquiry history failed:%d %v", pid, err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1},
		}, nil
	}
	log.Printf("after update inquiry_history status:%s %d", in.Oid, pid)
	var doctor, patient int64
	err = db.QueryRow("SELECT doctor, patient FROM inquiry_history WHERE id = ?", pid).Scan(&doctor, &patient)
	if err != nil {
		log.Printf("WxPayCB query inquiry history failed:%d %v", pid, err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1},
		}, nil
	}
	_, err = db.Exec("UPDATE relations SET hid = ?, flag = 1, status = 1 WHERE doctor = ? AND patient = ?", pid, doctor, patient)
	if err != nil {
		log.Printf("WxPayCB update relations failed:%d %v", pid, err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1},
		}, nil
	}
	log.Printf("after update relations flag, doctor:%d patient:%d %s",
		doctor, patient, in.Oid)
	_, err = db.Exec("UPDATE users SET hasrelation = 1, balance = balance + ?, totalfee = totalfee + ? WHERE uid = ?", in.Fee, in.Fee, doctor)
	if err != nil {
		log.Printf("WxPayCB update user hasrelation failed:%d %v", pid, err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1},
		}, nil
	}
	log.Printf("after update users hasrelation doctor:%d %s", doctor, in.Oid)
	openid, err := getUserOpenid(db, uid)
	if err == nil {
		ts := time.Now()
		ptime := fmt.Sprintf("%d年%d月%d日", ts.Year(), ts.Month(), ts.Day())
		money := fmt.Sprintf("人民币 %d元", in.Fee/100)
		var payInfos [4]string
		payInfos[0] = ptime
		payInfos[1] = "咨询费用"
		payInfos[2] = in.Oid
		payInfos[3] = money
		log.Printf("to sendPayWxMsg %s %v", openid, payInfos)
		sendPayWxMsg(db, openid, prepayid, payInfos)
	}
	phone, err := getUserPhone(db, tuid)
	if err == nil {
		util.SendPaySMS(phone)
	}
	util.PubRPCSuccRsp(w, "pay", "WxPayCB")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0},
	}, nil
}
