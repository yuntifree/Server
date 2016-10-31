package util

import (
	"database/sql"
	"log"
	"strconv"
)

const (
	//MaxListSize for page
	MaxListSize = 20
)

//UserInfo user base information
type UserInfo struct {
	UID, Sex                    int
	NickName, HeadURL, UserName string
}

//ExistPhone return whether phone exist
func ExistPhone(db *sql.DB, phone string) bool {
	var cnt int
	err := db.QueryRow("SELECT COUNT(uid) FROM user WHERE phone = ?", phone).Scan(&cnt)
	if err != nil {
		log.Printf("query failed:%v", err)
		return false
	}
	if cnt > 0 {
		return true
	}
	return false
}

//CheckToken verify user's token
func CheckToken(db *sql.DB, uid int64, token string, ctype int32) bool {
	var tk string
	var expire bool
	var err error
	if ctype == 0 {
		err = db.QueryRow("SELECT token, IF(etime > NOW(), false, true) FROM user WHERE uid = ?", uid).Scan(&tk, &expire)
	} else {
		err = db.QueryRow("SELECT skey, IF(expire_time > NOW(), false, true) FROM back_login WHERE uid = ?", uid).Scan(&tk, &expire)
	}
	if err != nil {
		log.Printf("query failed:%v", err)
		return false
	}
	if expire {
		log.Print("token expire")
		return false
	}
	if tk != token {
		log.Printf("token not match token:%s real:%s", token, tk)
		return false
	}
	return true
}

//ClearToken set token expire
func ClearToken(db *sql.DB, uid int64) {
	_, err := db.Exec("UPDATE user SET etime = NOW() WHERE uid = ?", uid)
	if err != nil {
		log.Printf("query failed:%v", err)
	}
}

//InitDB connect to rds
func InitDB() (*sql.DB, error) {
	return sql.Open("mysql", "access:^yunti9df3b01c$@tcp(rm-wz9sb2613092ki9xn.mysql.rds.aliyuncs.com:3306)/yunxing?charset=utf8")
}

//GetUserInfo select user info
func GetUserInfo(db *sql.DB, uinfo *UserInfo) (err error) {
	query := "SELECT uid, username, nickname, headurl, sex FROM user WHERE "
	if uinfo.UID != 0 {
		query += " uid = " + strconv.Itoa(uinfo.UID)
	} else if uinfo.UserName != "" {
		query += " username = '" + uinfo.UserName + "'"
	}
	err = db.QueryRow(query).Scan(&uinfo.UID, &uinfo.UserName, &uinfo.NickName, &uinfo.HeadURL, &uinfo.Sex)
	return
}
