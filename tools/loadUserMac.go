package main

import (
	"Server/util"
	"database/sql"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

func recordUserMac(db *sql.DB, phone, mac string, stype uint) {
	token := util.GenSalt()
	privdata := util.GenSalt()
	res, err := db.Exec("INSERT IGNORE INTO user(username, phone, token, private, term, ctime, atime, etime, bitmap) VALUES (?, ?, ?, ?, 2, NOW(), NOW(), DATE_ADD(NOW(), INTERVAL 30 DAY), ?)", phone, phone, token, privdata, 1<<stype)
	if err != nil {
		log.Printf("recordUserMac create user failed:%v", err)
		return
	}
	uid, err := res.LastInsertId()
	if err != nil {
		log.Printf("recordUserMac get user id failed:%v", err)
		return
	}
	_, err = db.Exec("INSERT INTO user_mac(uid, phone, mac, ctime, etime) VALUES (?, ?, ?, NOW(), DATE_ADD(NOW(), INTERVAL 30 DAY)) ON DUPLICATE KEY UPDATE uid = ?, phone = ?, etime = DATE_ADD(NOW(), INTERVAL 30 DAY)", uid, phone, mac, uid, phone)
	if err != nil {
		log.Printf("recordUserMac record mac failed:%v", err)
		return
	}
	return
}

func main() {
	db, err := util.InitDB(false)
	if err != nil {
		log.Printf("InitDB failed:%v", err)
		os.Exit(1)
	}
	phone := "18219201566"
	mac := "945330575f35"
	var stype uint
	stype = 1
	recordUserMac(db, phone, mac, stype)
}
