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

var db *sql.DB

var limit chan int
var wg sync.WaitGroup

func getAcnameType(acname string) uint {
	if util.IsWjjAcname(acname) {
		return 1
	}
	return 0
}

func kickOff(info *util.OnlineInfo) {
	log.Printf("kickOff online info:%v", info)
	stype := getAcnameType(info.Acname)
	flag := zte.Logout(info.Phone, info.Usermac, info.Userip, info.Acip, stype)
	if !flag {
		log.Printf("zte Logout failed:%v", info)
	}
}

func updateSubscribe(db *sql.DB, openid string) {
	_, err := db.Exec("UPDATE wx_conn SET subscribe = 1, stime = NOW() WHERE openid = ?", openid)
	if err != nil {
		log.Printf("updateSubscribe failed:%v", err)
	}
}

func checkUserOnline(db *sql.DB, accesstoken string, info *util.OnlineInfo) {
	defer wg.Done()
	limit <- 1
	log.Printf("checkUserOnline accesstoken:%s onlineinfo:%v", accesstoken, info)
	subscribe := util.CheckSubscribe(accesstoken, info.Openid)
	if !subscribe {
		kickOff(info)
	} else {
		updateSubscribe(db, info.Openid)
	}
	<-limit
}

func main() {
	limit = make(chan int, 64)
	db, err := util.InitDB(false)
	if err != nil {
		log.Printf("InitDB failed:%v", err)
		os.Exit(1)
	}
	db.SetMaxOpenConns(100)
	client := util.InitRedis()

	tasks := util.GetOnlineTask(client)
	accesstoken := util.GetAccessToken(db, 0)
	for i := 0; i < len(tasks); i++ {
		if tasks[i] == nil {
			continue
		}
		wg.Add(1)
		go checkUserOnline(db, accesstoken, tasks[i])
	}
	wg.Wait()
}
