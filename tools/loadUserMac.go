package main

import (
	"Server/util"
	"bufio"
	"database/sql"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

func recordUserMac(db *sql.DB, phone, mac string, stype uint) {
	token := util.GenSalt()
	privdata := util.GenSalt()
	res, err := db.Exec("INSERT INTO user(username, phone, token, private, term, ctime, atime, etime, bitmap) VALUES (?, ?, ?, ?, 2, NOW(), NOW(), DATE_ADD(NOW(), INTERVAL 30 DAY), ?) ON DUPLICATE KEY UPDATE bitmap = bitmap | ?", phone, phone, token, privdata, 1<<stype, 1<<stype)
	if err != nil {
		log.Printf("recordUserMac create user failed:%v", err)
		return
	}
	uid, err := res.LastInsertId()
	if err != nil {
		log.Printf("recordUserMac get user id failed:%v", err)
		return
	}
	if uid == 0 {
		err := db.QueryRow("SELECT uid FROM user WHERE username = ?", phone).Scan(&uid)
		if err != nil {
			log.Printf("recordUserMac get user id failed uid = 0, phone:%s mac:%s",
				phone, mac)
			return

		}
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
	f, err := os.Open("wg_phone_mac.csv")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	rd := bufio.NewReader(f)
	var idx int64
	for {
		idx++
		if idx%1000 == 0 {
			time.Sleep(1 * time.Second)
		}
		data, err := rd.ReadSlice('\n')
		if err != nil {
			break
		}
		line := string(data)
		arr := strings.Split(line, ";")
		phone := arr[0]
		mac := arr[1]
		phone = strings.TrimSpace(phone)
		mac = strings.TrimSpace(mac)
		if phone == "" || mac == "" {
			continue
		}
		log.Printf("phone:%s mac:%s", phone, mac)
		recordUserMac(db, phone, mac, 1)
	}
	return
}
