package main

import (
	"Server/proto/common"
	"Server/proto/inquiry"
	"Server/util"
	"Server/weixin"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"golang.org/x/net/context"
)

const (
	randrange = 1000000
	defCode   = 121949
)

func hasPatient(db *sql.DB, uid int64) int64 {
	var cnt int64
	err := db.QueryRow("SELECT COUNT(id) FROM patient WHERE uid = ? AND deleted = 0",
		uid).Scan(&cnt)
	if err != nil {
		return 0
	}
	if cnt > 0 {
		return 1
	}
	return 0
}

func (s *server) SubmitCode(ctx context.Context, in *inquiry.CodeRequest) (*inquiry.LoginReply, error) {
	log.Printf("SubmitCode request uid:%d code:%s", in.Head.Uid, in.Code)
	var openid, skey string
	var err error
	if in.Appid == weixin.DgyAppid {
		openid, skey, err = weixin.GetDgySession(in.Code)
	} else {
		openid, skey, err = weixin.GetInquirySession(in.Code)
	}
	if err != nil {
		log.Printf("SubmitCode GetSession failed:%v", err)
		return &inquiry.LoginReply{
			Head: &common.Head{Retcode: common.ErrCode_ILLEGAL_CODE}}, nil
	}
	log.Printf("openid:%s skey:%s", openid, skey)
	sid := util.GenSalt()
	var uid, role, hasrelation int64
	var phone, pass string
	err = db.QueryRow("SELECT u.uid, u.phone, u.draw_pass, u.role, u.hasrelation FROM users u, wx_openid x WHERE u.username = x.unionid AND x.openid = ?", openid).
		Scan(&uid, &phone, &pass, &role, &hasrelation)
	if err != nil {
		_, err = db.Exec("INSERT INTO wx_openid(openid, skey, sid, ctime) VALUES (?, ?, ?, NOW()) ON DUPLICATE KEY UPDATE skey = ?, sid = ?",
			openid, skey, sid, skey, sid)
		if err != nil {
			log.Printf("record failed:%v", err)
			return &inquiry.LoginReply{
				Head: &common.Head{Retcode: 1}}, nil
		}
	}
	log.Printf("uid:%d role:%d phone:%s hasrelation:%d", uid, role,
		phone, hasrelation)
	if uid == 0 {
		log.Printf("user not found, openid:%s", openid)
		return &inquiry.LoginReply{
			Head: &common.Head{Retcode: 0}, Flag: 0, Sid: sid}, nil
	}

	var hasphone int64
	if phone != "" {
		hasphone = 1
	}
	var haspatient int64
	if role == 0 {
		haspatient = hasPatient(db, uid)
	}
	var haspasswd int64
	if pass != "" {
		haspasswd = 1
	}
	token := util.GenSalt()
	_, err = db.Exec("UPDATE users SET token = ? WHERE uid = ?", token, uid)
	if err != nil {
		log.Printf("update token failed:%v", err)
		return &inquiry.LoginReply{
			Head: &common.Head{Retcode: 1}}, nil
	}

	return &inquiry.LoginReply{
		Head: &common.Head{Retcode: 0}, Flag: 1, Uid: uid, Token: token,
		Hasphone: hasphone, Role: role, Hasrelation: hasrelation,
		Haspatient: haspatient, Haspasswd: haspasswd, Phone: phone}, nil
}

func checkSign(skey, rawdata, signature string) bool {
	data := rawdata + skey
	sign := util.Sha1(data)
	return sign == signature
}

func extractUserInfo(skey, encrypted, iv string) (weixin.UserInfo, error) {
	var uinfo weixin.UserInfo
	dst, err := weixin.DecryptData(skey, encrypted, iv)
	if err != nil {
		log.Printf("aes decrypt failed skey:%s", skey)
		return uinfo, err
	}
	err = json.Unmarshal(dst, &uinfo)
	if err != nil {
		log.Printf("parse json failed:%s %v", string(dst), err)
		return uinfo, err
	}
	if uinfo.UnionId == "" {
		log.Printf("get unionid failed:%v", err)
		return uinfo, errors.New("get unionid failed")
	}
	return uinfo, nil
}

func (s *server) Login(ctx context.Context, in *inquiry.LoginRequest) (*inquiry.LoginReply, error) {
	log.Printf("Login request:%v", in)
	var skey, unionid, openid string
	err := db.QueryRow("SELECT skey, unionid, openid FROM wx_openid WHERE sid = ?", in.Sid).
		Scan(&skey, &unionid, &openid)
	if err != nil {
		log.Printf("illegal sid:%s", in.Sid)
		return &inquiry.LoginReply{
			Head: &common.Head{Retcode: 1}}, nil
	}
	if !checkSign(skey, in.Rawdata, in.Signature) {
		log.Printf("check signature failed sid:%s", in.Sid)
		return &inquiry.LoginReply{
			Head: &common.Head{Retcode: 1}}, nil
	}
	var uid, role, hasrelation int64
	var phone, pass string
	if unionid == "" { //has login
		uinfo, err := extractUserInfo(skey, in.Encrypteddata, in.Iv)
		if err != nil {
			log.Printf("extract user info failed sid:%s %v", in.Sid, err)
			return &inquiry.LoginReply{
				Head: &common.Head{Retcode: 1}}, nil
		}
		_, err = db.Exec("UPDATE wx_openid SET unionid = ? WHERE openid = ?",
			uinfo.UnionId, openid)
		if err != nil {
			log.Printf("update unionid failed:%v", err)
			return &inquiry.LoginReply{
				Head: &common.Head{Retcode: 1}}, nil
		}
		db.QueryRow("SELECT uid, phone, role, hasrelation, draw_pass FROM users WHERE username = ?",
			uinfo.UnionId).
			Scan(&uid, &phone, &role, &hasrelation, &pass)
		if uid == 0 {
			res, err := db.Exec("INSERT IGNORE INTO users(username, nickname, headurl, gender, ctime) VALUES (?, ?, ?, ?, NOW())",
				uinfo.UnionId, uinfo.NickName, uinfo.AvartarUrl, uinfo.Gender)
			if err != nil {
				log.Printf("create user failed:%v", err)
				return &inquiry.LoginReply{
					Head: &common.Head{Retcode: 1}}, nil
			}
			uid, _ = res.LastInsertId()
		}
	} else {
		db.QueryRow("SELECT uid FROM users WHERE username = ?", unionid).Scan(&uid)
	}
	if uid == 0 {
		log.Printf("select user failed sid:%s", in.Sid)
		return &inquiry.LoginReply{
			Head: &common.Head{Retcode: 1}}, nil
	}
	token := util.GenSalt()
	_, err = db.Exec("UPDATE users SET token = ? WHERE uid = ?", token, uid)
	if err != nil {
		log.Printf("update token failed sid:%s %v", in.Sid, err)
		return &inquiry.LoginReply{
			Head: &common.Head{Retcode: 1}}, nil
	}

	var hasphone int64
	if phone != "" {
		hasphone = 1
	}
	var haspatient int64
	if role == 0 {
		haspatient = hasPatient(db, uid)
	}
	var haspasswd int64
	if pass != "" {
		haspasswd = 1
	}

	return &inquiry.LoginReply{
		Head: &common.Head{Retcode: 0}, Uid: uid, Token: token,
		Hasphone: hasphone, Role: role, Hasrelation: hasrelation,
		Haspatient: haspatient, Haspasswd: haspasswd, Phone: phone}, nil
}

func (s *server) CheckToken(ctx context.Context, in *inquiry.TokenRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "inquiry", "CheckToken")
	log.Printf("CheckToken request:%+v", in)
	var token string
	err := db.QueryRow("SELECT token FROM users WHERE uid = ?", in.Head.Uid).
		Scan(&token)
	if err != nil {
		log.Printf("CheckToken query token failed:%v", err)
		return &common.CommReply{Head: &common.Head{Retcode: 1}}, nil
	}
	if token != in.Token {
		log.Printf("token not matched uid:%d token:%s-%s", in.Head.Uid,
			in.Token, token)
		return &common.CommReply{Head: &common.Head{Retcode: 1}}, nil
	}
	util.PubRPCSuccRsp(w, "inquiry", "CheckToken")
	return &common.CommReply{Head: &common.Head{Retcode: 0}}, nil
}

var errFrequency = errors.New("exceed frequency limit")

func getPhoneCode(phone string) error {
	log.Printf("request phone:%s", phone)
	var id, code, flag int
	err := db.QueryRow("SELECT pid, code, IF(NOW() > DATE_ADD(stime, INTERVAL 1 MINUTE), 0, 1) FROM phone_code WHERE phone = ? AND used = 0 AND etime > NOW() AND timestampdiff(second, stime, now()) < 300 ORDER BY pid DESC LIMIT 1",
		phone).Scan(&id, &code, &flag)
	if err != nil {
		code := util.Randn(randrange)
		_, err := db.Exec("INSERT INTO phone_code(phone, code, ctime, stime, etime) VALUES (?, ?, NOW(), NOW(), DATE_ADD(NOW(), INTERVAL 5 MINUTE))",
			phone, code)
		if err != nil {
			log.Printf("insert into phone_code failed:%v", err)
			return err
		}
		ret := util.SendHealthSMS(phone, fmt.Sprintf("%06d", code))
		if ret != nil {
			log.Printf("send sms failed:%d", ret)
			return errors.New("send sms failed")
		}
		return nil
	}

	if code > 0 && flag == 0 {
		ret := util.SendHealthSMS(phone, fmt.Sprintf("%06d", code))
		if ret != nil {
			log.Printf("send sms failed:%d", ret)
			return errors.New("send sms failed")
		}
		db.Exec("UPDATE phone_code SET stime = NOW() WHERE pid = ?", id)
		return nil
	} else if flag == 1 {
		return errFrequency
	}

	return errors.New("failed to send sms")
}

func (s *server) GetPhoneCode(ctx context.Context, in *inquiry.PhoneRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "inquiry", "GetPhoneCode")
	err := getPhoneCode(in.Phone)
	if err != nil {
		if err == errFrequency {
			return &common.CommReply{
				Head: &common.Head{
					Retcode: common.ErrCode_FREQUENCY_LIMIT}}, err
		}
		return &common.CommReply{Head: &common.Head{Retcode: 1}}, err
	}

	util.PubRPCSuccRsp(w, "inquiry", "GetPhoneCode")
	return &common.CommReply{Head: &common.Head{Retcode: 0}}, nil
}

func (s *server) CheckPhoneCode(ctx context.Context, in *inquiry.PhoneCodeRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "inquiry", "CheckPhoneCode")
	flag := checkPhoneCode(db, in.Phone, in.Code)
	if !flag {
		return &common.CommReply{
			Head: &common.Head{Retcode: common.ErrCode_CHECK_CODE}}, nil
	}
	util.PubRPCSuccRsp(w, "inquiry", "CheckPhoneCode")
	return &common.CommReply{Head: &common.Head{Retcode: 0}}, nil
}

func checkPhoneCode(db *sql.DB, phone string, code int64) bool {
	var id, ecode int64
	err := db.QueryRow("SELECT pid, code FROM phone_code WHERE phone = ? AND etime > NOW() ORDER BY pid DESC LIMIT 1", phone).
		Scan(&id, &ecode)
	if err != nil {
		log.Printf("checkPhoneCode get code failed:%s %v", phone, err)
		return false
	}
	if ecode != code {
		log.Printf("code not match, phone:%s code:%d - %d", phone, code, ecode)
		return false
	}
	_, err = db.Exec("UPDATE phone_code SET used = 1 WHERE pid = ?", id)
	if err != nil {
		log.Printf("checkPhoneCode set used failed, id:%d phone:%s", id,
			phone)
	}
	return true
}

func getPhoneRole(db *sql.DB, phone string) (doctor, role int64) {
	err := db.QueryRow("SELECT id FROM doctor WHERE phone = ?", phone).
		Scan(&doctor)
	if err != nil {
		return
	}
	if doctor > 0 {
		role = 1
	}
	return
}

func (s *server) BindPhone(ctx context.Context, in *inquiry.PhoneCodeRequest) (*inquiry.RoleReply, error) {
	util.PubRPCRequest(w, "inquiry", "BindPhone")
	if in.Code != defCode && !checkPhoneCode(db, in.Phone, in.Code) {
		log.Printf("BindPhone checkPhoneCode failed:%s %d", in.Phone, in.Code)
		return &inquiry.RoleReply{
			Head: &common.Head{
				Retcode: common.ErrCode_CHECK_CODE}}, nil
	}
	log.Printf("BindPhone request:%+v", in)

	doctor, role := getPhoneRole(db, in.Phone)
	_, err := db.Exec("UPDATE users SET phone = ?, role = ?, doctor = ? WHERE uid = ?",
		in.Phone, role, doctor, in.Head.Uid)
	if err != nil {
		log.Printf("BindPhone update user info failed:%d %v", in.Head.Uid,
			err)
		return &inquiry.RoleReply{
			Head: &common.Head{
				Retcode: 1}}, nil
	}

	if in.Tuid != 0 { //Bind Op
		err = addRelations(db, in.Head.Uid, in.Tuid)
		if err != nil {
			log.Printf("BindPhone addRelations tuid:%d failed:%v", in.Tuid, err)
		}
	}

	util.PubRPCSuccRsp(w, "inquiry", "BindPhone")
	return &inquiry.RoleReply{Head: &common.Head{Retcode: 0}, Role: role}, nil
}
