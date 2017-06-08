package main

import (
	"Server/proto/common"
	"Server/proto/inquiry"
	"Server/util"
	"database/sql"
	"errors"
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

func addPatient(db *sql.DB, uid int64, info *inquiry.PatientInfo) (int64, error) {
	res, err := db.Exec("INSERT INTO patient(uid, name, phone, mcard, ctime) VALUES (?, ?, ?, ?, NOW())",
		uid, info.Name, info.Phone, info.Mcard)
	if err != nil {
		log.Printf("addPatient insert failed:%v", err)
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Printf("addPatient get insert id failed:%v", err)
		return 0, err
	}
	return id, nil
}

func (s *server) AddPatient(ctx context.Context, in *inquiry.PatientRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "inquiry", "AddPatient")
	id, err := addPatient(db, in.Head.Uid, in.Info)
	if err != nil {
		log.Printf("gePatients failed:%d %v", in.Head.Uid, err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	util.PubRPCSuccRsp(w, "inquiry", "AddPatient")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Id: id}, nil
}

func modPatient(db *sql.DB, uid int64, info *inquiry.PatientInfo) error {
	var euid int64
	err := db.QueryRow("SELECT uid FROM patient WHERE id = ?", info.Id).
		Scan(&euid)
	if err != nil {
		log.Printf("modPatient get uid failed:%v", err)
		return err
	}
	if uid != euid {
		log.Printf("not match uid:%d - %d", uid, euid)
		return errors.New("uid not matched")
	}
	_, err = db.Exec("UPDATE patient SET name = ?, phone = ?, mcard = ?, deleted = ? WHERE id = ?",
		info.Name, info.Phone, info.Mcard, info.Deleted, info.Id)
	if err != nil {
		log.Printf("modPatient update failed:%d %v", info.Id, err)
		return err
	}
	return nil
}

func (s *server) ModPatient(ctx context.Context, in *inquiry.PatientRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "inquiry", "AddPatient")
	err := modPatient(db, in.Head.Uid, in.Info)
	if err != nil {
		log.Printf("gePatients failed:%d %v", in.Head.Uid, err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	util.PubRPCSuccRsp(w, "inquiry", "ModPatient")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}
