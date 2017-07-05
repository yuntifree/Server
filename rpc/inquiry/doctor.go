package main

import (
	"Server/aliyun"
	"Server/proto/common"
	"Server/proto/inquiry"
	"Server/util"
	"Server/weixin"
	"database/sql"
	"errors"
	"log"

	"golang.org/x/net/context"
)

func getDoctorInfo(db *sql.DB, uid int64) (*inquiry.DoctorInfo, error) {
	var role, doctor int64
	err := db.QueryRow("SELECT role, doctor FROM users WHERE uid = ?", uid).
		Scan(&role, &doctor)
	if err != nil {
		log.Printf("getDoctorInfo query role failed:%d %v", uid, err)
		return nil, err
	}
	if role == 0 || doctor == 0 {
		log.Printf("getDoctorInfo not doctor, uid:%d role:%d doctor:%d",
			uid, role, doctor)
		return nil, errors.New("not doctor")
	}
	var info inquiry.DoctorInfo
	err = db.QueryRow("SELECT name, headurl, title, hospital, department, fee FROM doctor WHERE id = ?", doctor).
		Scan(&info.Name, &info.Headurl, &info.Title, &info.Hospital,
			&info.Department, &info.Fee)
	if err != nil {
		log.Printf("getDoctorInfo get info failed:%d %v", uid, err)
		return nil, err
	}
	return &info, nil
}

func (s *server) GetDoctorInfo(ctx context.Context, in *common.CommRequest) (*inquiry.DoctorInfoReply, error) {
	util.PubRPCRequest(w, "inquiry", "GetDoctorInfo")
	info, err := getDoctorInfo(db, in.Id)
	if err != nil {
		log.Printf("getDoctorInfo failed:%d %v", in.Id, err)
		return &inquiry.DoctorInfoReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	/*
		if in.Head.Uid != in.Id {
			info.Fee = (int64(float64(info.Fee)*feeRate) / 100) * 100
		}
	*/
	util.PubRPCSuccRsp(w, "inquiry", "GetDoctorInfo")
	return &inquiry.DoctorInfoReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Info: info}, nil
}

func getPatientInfo(db *sql.DB, uid, tuid int64) (*inquiry.PatientInfo, error) {
	var pid int64
	err := db.QueryRow("SELECT pid FROM inquiry_history WHERE doctor = ? AND patient = ? ORDER BY id DESC LIMIT 1", uid, tuid).Scan(&pid)
	if err != nil {
		log.Printf("getPatientInfo get pid failed:%d %d %v", uid, tuid,
			err)
		return nil, err
	}
	var info inquiry.PatientInfo
	err = db.QueryRow("SELECT name, mcard, phone, gender, age FROM patient WHERE id = ?", pid).
		Scan(&info.Name, &info.Mcard, &info.Phone, &info.Gender,
			&info.Age)
	if err != nil {
		log.Printf("getPatientInfo query failed:%d %v", uid, err)
		return nil, err
	}
	return &info, nil
}

func (s *server) GetPatientInfo(ctx context.Context, in *common.CommRequest) (*inquiry.PatientInfoReply, error) {
	util.PubRPCRequest(w, "inquiry", "GetPatientInfo")
	info, err := getPatientInfo(db, in.Head.Uid, in.Id)
	if err != nil {
		log.Printf("getPatientInfo failed:%d %v", in.Id, err)
		return &inquiry.PatientInfoReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	util.PubRPCSuccRsp(w, "inquiry", "GetPatientInfo")
	return &inquiry.PatientInfoReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Info: info}, nil
}

func genQRCodeImg(db *sql.DB, path string, width int64) (string, error) {
	accessToken, err := weixin.RefreshAccessToken(db, weixin.InquiryAppid)
	if err != nil {
		log.Printf("genQRCodeImg RefreshAccessToken failed:%v", err)
		return "", err
	}
	qrcode, err := weixin.CreateQRCode(accessToken, path, width)
	if err != nil {
		log.Printf("genQRCodeImg CreateQRCode failed:%v", err)
		return "", err
	}
	filename := util.GenUUID() + ".jpg"
	flag := aliyun.UploadOssImg(filename, qrcode)
	if !flag {
		log.Printf("genQRCodeImg UploadOssImg failed:%v", err)
		return "", err
	}
	url := aliyun.GenOssImgURL(filename)
	return url, nil
}

func (s *server) GetQRCode(ctx context.Context, in *inquiry.QRCodeRequest) (*inquiry.QRCodeReply, error) {
	util.PubRPCRequest(w, "inquiry", "GetQRCode")
	var img string
	err := db.QueryRow("SELECT img FROM qrcode WHERE width = ? AND path = ?",
		in.Width, in.Path).Scan(&img)
	if err != nil || img == "" {
		img, err = genQRCodeImg(db, in.Path, in.Width)
		if err != nil {
			log.Printf("GetQRCode getQRCodeImg failed:%v", err)
			return &inquiry.QRCodeReply{
				Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
		}
		_, err := db.Exec("INSERT INTO qrcode(path, img, width, ctime) VALUES (?, ?, ?,NOW())", in.Path, img, in.Width)
		if err != nil {
			log.Printf("GetQRCode record new qrcode failed:%v", err)
		}
	}

	util.PubRPCSuccRsp(w, "inquiry", "GetQRCode")
	return &inquiry.QRCodeReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Url: img}, nil
}

func (s *server) SetFee(ctx context.Context, in *inquiry.FeeRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "inquiry", "SetFee")
	var role, doctor int64
	err := db.QueryRow("SELECT role, doctor FROM users WHERE uid = ?",
		in.Head.Uid).Scan(&role, &doctor)
	if err != nil {
		log.Printf("SetFee get user role failed, uid:%d %v", in.Head.Uid, err)
		return &common.CommReply{Head: &common.Head{Retcode: 1}}, nil
	}
	if role == 0 || doctor == 0 {
		log.Printf("SetFee not doctor uid:%d", in.Head.Uid)
		return &common.CommReply{Head: &common.Head{Retcode: 1}}, nil
	}
	_, err = db.Exec("UPDATE doctor SET fee = ? WHERE id = ?", in.Fee, doctor)
	if err != nil {
		log.Printf("SetFee update failed:%d %v", in.Head.Uid, err)
		return &common.CommReply{Head: &common.Head{Retcode: 1}}, nil
	}
	util.PubRPCSuccRsp(w, "inquiry", "SetFee")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0}}, nil
}
