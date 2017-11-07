package main

import (
	"Server/proto/common"
	"Server/proto/config"
	"Server/util"
	"database/sql"
	"log"

	"golang.org/x/net/context"
)

func (s *server) GetWxMpInfo(ctx context.Context, in *common.CommRequest) (*config.WxMpReply, error) {
	util.PubRPCRequest(w, "config", "GetWxMpInfo")
	infos := getWxMpInfo(db)
	util.PubRPCSuccRsp(w, "config", "GetWxMpInfo")
	return &config.WxMpReply{
		Head:  &common.Head{Retcode: 0, Uid: in.Head.Uid},
		Infos: infos}, nil
}

func getWxMpInfo(db *sql.DB) []*config.WxMpInfo {
	rows, err := db.Query(`SELECT id, appid, shopid, secret, title FROM 
	wx_appinfo WHERE deleted = 0`)
	if err != nil {
		log.Printf("getWxMpInfo query failed:%v", err)
		return nil
	}
	defer rows.Close()
	var infos []*config.WxMpInfo
	for rows.Next() {
		var info config.WxMpInfo
		err = rows.Scan(&info.Id, &info.Appid, &info.Shopid,
			&info.Secret, &info.Title)
		if err != nil {
			continue
		}
		infos = append(infos, &info)
	}
	return infos
}

func (s *server) AddWxMpInfo(ctx context.Context, in *config.WxMpRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "config", "AddWxMpInfo")
	id, err := addWxMpInfo(db, in.Info)
	if err != nil {
		log.Printf("AddWxMpInfo failed:%v", err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	util.PubRPCSuccRsp(w, "config", "AddWxMpInfo")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid},
		Id:   id}, nil
}

func addWxMpInfo(db *sql.DB, info *config.WxMpInfo) (int64, error) {
	res, err := db.Exec(`INSERT INTO wx_appinfo(appid, shopid, secret,
	title, authurl, ctime) VALUES (?, ?, ?, ?, 'http://wx.yunxingzh.com/auth',
	NOW())`, info.Appid, info.Shopid, info.Secret, info.Title)
	if err != nil {
		log.Printf("addWxMpInfo insert failed:%+v %v", info, err)
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Printf("addWxMpInfo get insert id failed:%v", err)
		return 0, err
	}
	return id, nil
}
