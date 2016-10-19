package util

import (
	"database/sql"
	"log"
)

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
func CheckToken(db *sql.DB, uid int64, token string) bool {
	var tk string
	var expire bool
	err := db.QueryRow("SELECT token, IF(etime > NOW(), false, true) FROM user WHERE uid = ?", uid).Scan(&tk, &expire)
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
