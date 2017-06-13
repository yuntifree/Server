package main

import (
	"Server/proto/common"
	"Server/proto/inquiry"
	"Server/util"
	"log"

	"golang.org/x/net/context"
)

func (s *server) GetWallet(ctx context.Context, in *common.CommRequest) (*inquiry.WalletReply, error) {
	util.PubRPCRequest(w, "inquiry", "GetWallet")
	var balance, total, draw, totaldraw int64
	err := db.QueryRow("SELECT balance, total, draw, totaldraw FROM users WHERE uid = ?", in.Head.Uid).
		Scan(&balance, &total, &draw, &totaldraw)
	if err != nil {
		log.Printf("GetWallet query failed:%d %v", in.Head.Uid, err)
		return &inquiry.WalletReply{Head: &common.Head{Retcode: 1}}, nil

	}
	return &inquiry.WalletReply{Head: &common.Head{Retcode: 0},
		Balance: balance, Total: total, Draw: draw, Totaldraw: totaldraw}, nil
}

func (s *server) ApplyDraw(ctx context.Context, in *inquiry.DrawRequest) (*common.CommReply, error) {
	log.Printf("ApplyDraw request:%+v", in)
	util.PubRPCRequest(w, "inquiry", "GetWallet")
	var balance int64
	err := db.QueryRow("SELECT balance FROM users WHERE uid = ?", in.Head.Uid).
		Scan(&balance)
	if err != nil {
		log.Printf("ApplyDraw query failed:%d %v", in.Head.Uid, err)
		return &common.CommReply{Head: &common.Head{Retcode: 1}}, nil
	}
	if balance < in.Fee {
		log.Printf("ApplyDraw insufficient balance, uid:%d %d-%d",
			in.Head.Uid, balance, in.Fee)
		return &common.CommReply{
			Head: &common.Head{
				Retcode: common.ErrCode_INSUFFICIENT_BALANCE}}, nil
	}
	_, err = db.Exec("UPDATE users SET balance = IF(balance > ?, balance - ?, 0), draw = draw + ? WHERE uid = ?",
		in.Fee, in.Fee, in.Fee, in.Head.Uid)
	if err != nil {
		log.Printf("ApplyDraw update balance failed, uid:%d %d", in.Head.Uid,
			in.Fee)
		return &common.CommReply{Head: &common.Head{Retcode: 1}}, nil
	}
	_, err = db.Exec("INSERT INTO draw_history(uid, fee, ctime) VALUES (?, ?, NOW())",
		in.Head.Uid, in.Fee)
	if err != nil {
		log.Printf("ApplyDraw record failed, uid:%d %d", in.Head.Uid,
			in.Fee)
		return &common.CommReply{Head: &common.Head{Retcode: 1}}, nil
	}
	return &common.CommReply{Head: &common.Head{Retcode: 0}}, nil
}
