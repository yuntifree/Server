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
