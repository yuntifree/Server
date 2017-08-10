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

type Task struct {
	Id   int64
	Info util.OnlineInfo
}

func main() {
	limit = make(chan int, 64)
	db, err := util.InitDB(false)
	if err != nil {
		log.Printf("InitDB failed:%v", err)
		os.Exit(1)
	}
	db.SetMaxOpenConns(10)

	tasks := getTasks(db)

	accesstoken := util.GetAccessToken(db, 2)
	for i := 0; i < len(tasks); i++ {
		if tasks[i] == nil {
			continue
		}
		wg.Add(1)
		go checkUserOnline(db, accesstoken, tasks[i])
	}
	wg.Wait()
}

func getTasks(db *sql.DB) []*Task {
	rows, err := db.Query(`SELECT w.id, w.openid, w.acname, w.acip, w.usermac, 
	w.userip, o.phone FROM wx_conn_info w, online_status o 
	WHERE w.appid = 'wx14606cf1ccfb0695' AND w.usermac = o.mac 
	AND w.etime > NOW() AND w.etime < DATE_ADD(NOW(), INTERVAL 10 MINUTE) 
	AND w.subscribe = 0 AND o.etime > NOW()`)
	if err != nil {
		log.Printf("getTasks failed:%v", err)
		return nil
	}
	defer rows.Close()
	var tasks []*Task
	for rows.Next() {
		var t Task
		err = rows.Scan(&t.Id, &t.Info.Openid, &t.Info.Acname,
			&t.Info.Acip, &t.Info.Usermac,
			&t.Info.Phone)
		if err != nil {
			log.Printf("getTasks scan failed:%v", err)
			continue
		}
		tasks = append(tasks, &t)
	}
	return tasks
}

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

func updateSubscribe(db *sql.DB, id int64) {
	_, err := db.Exec("UPDATE wx_conn_info SET subscribe = 1, stime = NOW() WHERE id = ?", id)
	if err != nil {
		log.Printf("updateSubscribe failed:%d %v", id, err)
	}
}

func checkUserOnline(db *sql.DB, accesstoken string, t *Task) {
	defer wg.Done()
	limit <- 1
	log.Printf("checkUserOnline accesstoken:%s onlineinfo:%+v", accesstoken, t)
	subscribe := util.CheckSubscribe(accesstoken, t.Info.Openid)
	if !subscribe {
		kickOff(&t.Info)
	} else {
		updateSubscribe(db, t.Id)
	}
	<-limit
}
