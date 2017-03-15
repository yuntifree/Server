package main

import (
	"Server/util"
	"Server/zte"
	"log"
	"os"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func diff(end, start string) int64 {
	t1, err := time.Parse("2006-01-02 15:04:05", start)
	if err != nil {
		log.Printf("time parse failed:%v", err)
		return 0
	}
	t2, err := time.Parse("2006-01-02 15:04:05", end)
	if err != nil {
		log.Printf("time parse failed:%v", err)
		return 0
	}
	return t2.Unix() - t1.Unix()
}

func main() {
	if len(os.Args) < 2 {
		log.Printf("Usage:%s [uid]", os.Args[0])
		os.Exit(1)
	}
	db, err := util.InitDB(false)
	if err != nil {
		log.Printf("InitDB failed:%v", err)
		os.Exit(1)
	}
	uid, err := strconv.ParseInt(os.Args[1], 10, 64)
	if err != nil {
		log.Printf("illegal uid:%s %v", os.Args[1], err)
		os.Exit(1)
	}
	var phone, regtime, today string
	err = db.QueryRow("SELECT phone, DATE(ctime), DATE(NOW()) FROM user WHERE uid = ?", uid).Scan(&phone, &regtime, &today)
	if err != nil {
		log.Printf("query phone failed:%v", err)
		os.Exit(1)
	}
	log.Printf("phone:%s, regtime:%s today:%s", phone, regtime, today)

	t1, err := time.Parse("2006-01-02", regtime)
	if err != nil {
		log.Printf("parse regtime failed:%v", err)
		os.Exit(1)
	}
	t2, err := time.Parse("2006-01-02", today)
	if err != nil {
		log.Printf("parse today failed:%v", err)
		os.Exit(1)
	}

	var duration int64
	var total int
	var times int
	for t1.Before(t2) {
		start := t1.Format(util.TimeFormat)
		end := t1.Add(24 * time.Hour).Format(util.TimeFormat)
		log.Printf("start:%s end:%s", start, end)
		t1 = t1.Add(24 * time.Hour)

		records := zte.GetOnlineRecords(phone, start, end)
		for i := 0; i < len(records); i++ {
			traffic, _ := strconv.Atoi(records[i].Traffic)
			log.Printf("record aid:%d stime:%s etime:%s traffic:%d", records[i].Aid, records[i].Start, records[i].End, traffic)
			_, err := db.Exec("INSERT INTO user_record(uid, aid, stime, etime, traffic) VALUES (?, ?, ?, ?,?)", uid, records[i].Aid, records[i].Start, records[i].End, traffic)
			if err != nil {
				log.Printf("insert failed:%v", err)
				continue
			}
			duration += diff(records[i].End, records[i].Start)
			total += traffic
		}
		times += len(records)
	}
	_, err = db.Exec("UPDATE user SET times = times + ?, duration = duration + ?, traffic = traffic + ? WHERE uid = ?", times, duration, total, uid)
	if err != nil {
		log.Printf("update user info failed:%v", err)
	}
	log.Printf("uid:%d username:%s times:%d duration:%d traffic:%d\n", uid, phone, times, duration, total)
}
