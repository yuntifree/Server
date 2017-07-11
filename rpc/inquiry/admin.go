package main

import (
	"Server/proto/common"
	"Server/proto/inquiry"
	"Server/util"
	"fmt"
	"log"

	"golang.org/x/net/context"
)

func (s *server) DelUser(ctx context.Context, in *inquiry.PhoneRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "inquiry", "DelUser")
	rows, err := db.Query("SELECT username FROM users WHERE phone = ?",
		in.Phone)
	if err != nil {
		log.Printf("DelUser get user name failed:%s %v", in.Phone,
			err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	defer rows.Close()
	var ids string
	for rows.Next() {
		var name string
		err = rows.Scan(&name)
		if err != nil {
			log.Printf("DelUser scan failed:%v", err)
			continue
		}
		ids += fmt.Sprintf("'%s',", name)
	}
	ids += "'0'"
	_, err = db.Exec("DELETE FROM wx_openid WHERE unionid IN (?)", ids)
	if err != nil {
		log.Printf("DelUser delete from wx_openid failed:%s %v", ids, err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	_, err = db.Exec("DELETE FROM users WHERE phone = ?", in.Phone)
	if err != nil {
		log.Printf("DelUser delete from users failed:%s %v",
			in.Phone, err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	util.PubRPCSuccRsp(w, "inquiry", "DelUser")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func (s *server) SetDoctor(ctx context.Context, in *inquiry.PhoneRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "inquiry", "SetDoctor")
	_, err := db.Exec("UPDATE users SET role = 1, doctor = 1 WHERE phone = ?", in.Phone)
	if err != nil {
		log.Printf("SetDoctor query failed:%s %v", in.Phone, err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	util.PubRPCSuccRsp(w, "inquiry", "SetDoctor")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}
