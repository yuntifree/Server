package main

import (
	"Server/util"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func getActiveUser(db *sql.DB) int64 {
	var total int64
	err := db.QueryRow("SELECT COUNT(uid) FROM user WHERE atime > DATE_SUB(CURDATE(), INTERVAL 1 DAY)").Scan(&total)
	if err != nil {
		log.Printf("getActiveUser failed:%v", err)
	}
	return total
}

func getRegisterUser(db *sql.DB) int64 {
	var total int64
	err := db.QueryRow("SELECT COUNT(uid) FROM user WHERE ctime >= DATE_SUB(CURDATE(), INTERVAL 1 DAY) AND ctime < CURDATE()").Scan(&total)
	if err != nil {
		log.Printf("getRegisterUser failed:%v", err)
	}
	return total
}

func main() {
	db, err := util.InitDB(false)
	if err != nil {
		log.Printf("InitDB failed:%v", err)
		os.Exit(1)
	}

	active := getActiveUser(db)
	register := getRegisterUser(db)
	_, err = db.Exec("INSERT IGNORE INTO user_stat(active, register, ctime) VALUES(?, ?, DATE_SUB(CURDATE(), INTERVAL 1 DAY))", active, register)
	if err != nil {
		log.Printf("insert failed:%v", err)
	}
	msg := fmt.Sprintf("%s user stat active:%d register:%d",
		time.Now().Format(util.TimeFormat), active, register)
	util.SendCronMail(msg)
	return
}
