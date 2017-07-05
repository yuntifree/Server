package main

import (
	"Server/proto/common"
	"Server/util"
	"database/sql"
	"log"

	"golang.org/x/net/context"
)

func addRelations(db *sql.DB, patient, doctor int64) error {
	_, err := db.Exec("INSERT INTO relations(doctor, patient, ctime) VALUES (?, ?, NOW()) ON DUPLICATE KEY UPDATE deleted = 0",
		doctor, patient)
	if err != nil {
		log.Printf("addRelations failed:%d %d %v", doctor, patient)
		return err
	}
	_, err = db.Exec("UPDATE users SET hasrelation = 1 WHERE uid = ?", patient)
	if err != nil {
		log.Printf("BindOp update user relation failed:%d %v", patient,
			err)
	}
	return nil
}

func removeRelations(db *sql.DB, patient, doctor int64) error {
	_, err := db.Exec("UPDATE relations SET deleted = 1 WHERE doctor = ? AND patient = ?",
		doctor, patient)
	if err != nil {
		log.Printf("removeRelations failed:%d %d %v", doctor, patient)
		return err
	}
	var cnt int64
	err = db.QueryRow("SELECT COUNT(id) FROM relations WHERE patient = ? AND deleted = 0", patient).Scan(&cnt)
	if err != nil {
		log.Printf("BindOp get relations failed:%v", err)
	}
	if cnt == 0 {
		_, err = db.Exec("UPDATE users SET hasrelation = 0 WHERE uid = ?", patient)
		if err != nil {
			log.Printf("BindOp update user relation failed:%d %v", patient,
				err)
		}
	}

	err = db.QueryRow("SELECT COUNT(id) FROM relations WHERE doctor = ? AND deleted = 0 AND flag = 1", doctor).Scan(&cnt)
	if err != nil {
		log.Printf("BindOp get relations failed:%v", err)
	}
	if cnt == 0 {
		_, err = db.Exec("UPDATE users SET hasrelation = 0 WHERE uid = ?", doctor)
		if err != nil {
			log.Printf("BindOp update user relation failed:%d %v", patient,
				err)
		}
	}
	return nil
}

func (s *server) BindOp(ctx context.Context, in *common.CommRequest) (*common.CommReply, error) {
	log.Printf("BindOp request:%+v", in)
	util.PubRPCRequest(w, "inquiry", "BindOp")
	var err error
	if in.Type == 0 {
		err = addRelations(db, in.Head.Uid, in.Id)
	} else {
		err = removeRelations(db, in.Head.Uid, in.Id)
	}
	if err != nil {
		log.Printf("BindOp op failed:%v", err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1}}, nil
	}
	util.PubRPCSuccRsp(w, "inquiry", "BindOp")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0}}, nil
}
