package main

import (
	"Server/util"
	"Server/zte"
	"database/sql"
	"log"
	"os"
	"sync"

	_ "github.com/go-sql-driver/mysql"
)

type portalInfo struct {
	Phone   string
	Usermac string
	Userip  string
	Acip    string
	Stype   uint
}

var db *sql.DB

var limit chan int
var wg sync.WaitGroup

func getAcnameType(acname string) uint {
	if util.IsWjjAcname(acname) {
		return 1
	}
	return 0
}

func kickOff(db *sql.DB, usermac, acname string) {
	log.Printf("kickOff usermac:%s acname:%s", usermac, acname)
	var info portalInfo
	err := db.QueryRow("SELECT phone, ip, acip FROM online_status WHERE mac = ? AND etime > NOW()", usermac).Scan(&info.Phone, &info.Userip, &info.Acip)
	if err != nil {
		log.Printf("kickOff query failed:%v", err)
		return
	}
	info.Usermac = usermac
	info.Stype = getAcnameType(acname)
	log.Printf("kickOff usermac:%s phone:%s", usermac, info.Phone)
	flag := zte.Logout(info.Phone, info.Usermac, info.Userip, info.Acip, info.Stype)
	if !flag {
		log.Printf("zte Logout failed:%v", info)
	}
}

func checkUserOnline(db *sql.DB, openid, usermac, acname string) {
	defer wg.Done()
	limit <- 1
	log.Printf("checkUserOnline openid:%s usermac:%s acname:%s", openid, usermac, acname)
	accesstoken := util.GetAccessToken(db, 0)
	subscribe := util.CheckSubscribe(accesstoken, openid)
	if !subscribe {
		kickOff(db, usermac, acname)
	}
	<-limit
}

func main() {
	limit = make(chan int, 8)
	db, err := util.InitDB(false)
	if err != nil {
		log.Printf("InitDB failed:%v", err)
		os.Exit(1)
	}

	rows, err := db.Query("SELECT openid, usermac, acname FROM wx_conn WHERE etime > NOW() AND acname = 'AC_120_A_06'")
	if err != nil {
		log.Printf("query failed:%v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var openid, usermac, acname string
		err = rows.Scan(&openid, &usermac, &acname)
		if err != nil {
			log.Printf("scan failed:%v", err)
			continue
		}
		wg.Add(1)
		go checkUserOnline(db, openid, usermac, acname)
	}
	wg.Wait()
}
