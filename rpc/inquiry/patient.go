package main

import (
	"Server/proto/common"
	"Server/proto/inquiry"
	"Server/util"
	"database/sql"
	"errors"
	"fmt"
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
	util.PubRPCSuccRsp(w, "inquiry", "GetPatients")
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
		log.Printf("addPatients failed:%d %v", in.Head.Uid, err)
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

func getLastestChat(db *sql.DB, uid, tuid int64) *inquiry.ChatInfo {
	var info inquiry.ChatInfo
	err := db.QueryRow("SELECT id, uid, tuid, type, content, ctime FROM chat WHERE ((uid = ? AND tuid = ?) OR (uid = ? AND tuid = ?)) ORDER BY id DESC LIMIT 1",
		uid, tuid, tuid, uid).
		Scan(&info.Id, &info.Uid, &info.Tuid, &info.Type, &info.Content,
			&info.Ctime)
	if err != nil {
		log.Printf("getLastestChat query failed:%v", err)
		return nil
	}
	return &info
}

func getDoctors(db *sql.DB, uid, seq, num int64) ([]*inquiry.Doctor, error) {
	query := "SELECT r.id, r.doctor, r.flag, r.status, d.name, d.headurl, d.hospital, d.department, d.title FROM relations r, doctor d, users u WHERE r.doctor = u.uid AND u.doctor = d.id AND r.deleted = 0 AND d.deleted = 0 AND u.deleted = 0"
	query += fmt.Sprintf(" AND r.patient = %d", uid)
	if seq != 0 {
		query += fmt.Sprintf(" AND r.id < %d", seq)
	}
	query += fmt.Sprintf(" ORDER BY r.id DESC LIMIT %d", num)
	log.Printf("getDoctors query:%s", query)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("getDoctors query failed:%d %v", uid, err)
		return nil, err
	}
	defer rows.Close()
	var infos []*inquiry.Doctor
	for rows.Next() {
		var doc inquiry.Doctor
		var info inquiry.DoctorInfo
		err = rows.Scan(&doc.Id, &doc.Uid, &doc.Flag, &doc.Status, &info.Name,
			&info.Headurl, &info.Hospital, &info.Department,
			&info.Title)
		if err != nil {
			log.Printf("getDoctors scan failed:%d %v", uid, err)
			continue
		}
		doc.Doctor = &info
		if doc.Flag > 0 {
			chat := getLastestChat(db, uid, doc.Uid)
			if chat != nil {
				doc.Chat = chat
			}
		}
		infos = append(infos, &doc)
	}
	return infos, nil
}

func (s *server) GetDoctors(ctx context.Context, in *common.CommRequest) (*inquiry.DoctorsReply, error) {
	util.PubRPCRequest(w, "inquiry", "GetDoctors")
	infos, err := getDoctors(db, in.Head.Uid, in.Seq, in.Num)
	if err != nil {
		log.Printf("getDoctors failed:%d %v", in.Head.Uid, err)
		return &inquiry.DoctorsReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	util.PubRPCSuccRsp(w, "inquiry", "GetDoctors")
	var hasmore int64
	if len(infos) >= int(in.Num) {
		hasmore = 1
	}
	return &inquiry.DoctorsReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Infos: infos,
		Hasmore: hasmore}, nil
}
