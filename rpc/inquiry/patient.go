package main

import (
	"Server/proto/common"
	"Server/proto/inquiry"
	"Server/util"
	"database/sql"
	"log"

	"golang.org/x/net/context"
)

func getPatients(db *sql.DB, uid int64) ([]*inquiry.PatientInfo, error) {
	rows, err := db.Query("SELECT id, name, phone, mcard FROM patient WHERE deleted = 0 AND uid = ?", uid)
	if err != nil {
		log.Printf("getPatients query failed:%d %v", uid, err)
		return nil, err
	}
	defer rows.Close()
	var infos []*inquiry.PatientInfo
	for rows.Next() {
		var info inquiry.PatientInfo
		err = rows.Scan(&info.Id, &info.Name, &info.Phone, &info.Mcard)
		if err != nil {
			log.Printf("getPatients scan failed:%v", err)
			continue
		}
		infos = append(infos, &info)
	}
	return infos, nil
}

func (s *server) GetPatients(ctx context.Context, in *common.CommRequest) (*inquiry.PatientsReply, error) {
	util.PubRPCRequest(w, "inquiry", "GetPatients")
	infos, err := getPatients(db, in.Head.Uid)
	if err != nil {
		log.Printf("gePatients failed:%d %v", in.Id, err)
		return &inquiry.PatientsReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	util.PubRPCSuccRsp(w, "inquiry", "GetDoctorInfo")
	return &inquiry.PatientsReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Infos: infos}, nil
}
