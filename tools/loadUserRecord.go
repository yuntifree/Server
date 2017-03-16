package main

import (
	"Server/util"
	"Server/zte"
	"fmt"
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

func genTableName() string {
	now := time.Now()
	year, month, _ := now.Date()
	return fmt.Sprintf("user_record_%4d%02d", year, month)
}

func main() {
	db, err := util.InitDB(false)
	if err != nil {
		log.Printf("InitDB failed:%v", err)
		os.Exit(1)
	}
	table := genTableName()
	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s LIKE user_record", table)
	_, err = db.Exec(query)
	if err != nil {
		log.Printf("create table failed:%v", err)
		os.Exit(1)
	}
	rows, err := db.Query("SELECT uid, username, CURDATE(), DATE_SUB(CURDATE(), INTERVAL 1 DAY) FROM user WHERE atime > DATE_SUB(CURDATE(), INTERVAL 1 DAY)")
	if err != nil {
		log.Fatalf("query failed:%v", err)
	}

	defer rows.Close()
	var cnt int
	for rows.Next() {
		cnt++
		var uid int
		var username string
		var end, start string
		err = rows.Scan(&uid, &username, &end, &start)
		if err != nil {
			log.Printf("scan rows failed:%v", err)
			continue
		}
		records := zte.GetOnlineRecords(username, start, end)
		var duration int64
		if len(records) > 0 {
			total := 0
			for i := 0; i < len(records); i++ {
				traffic, _ := strconv.Atoi(records[i].Traffic)
				_, err := db.Exec("INSERT INTO "+table+"(uid, aid, stime, etime, traffic) VALUES (?, ?, ?, ?,?)",
					uid, records[i].Aid, records[i].Start, records[i].End, traffic)
				if err != nil {
					log.Printf("insert failed:%v", err)
					continue
				}
				duration += diff(records[i].End, records[i].Start)
				total += traffic
			}
			_, err := db.Exec("UPDATE user SET times = times + ?, duration = duration + ?, traffic = traffic + ? WHERE uid = ?", len(records), duration, total, uid)
			if err != nil {
				log.Printf("update user info failed:%v", err)
			}
			log.Printf("uid:%d username:%s times:%d duration:%d traffic:%d\n", uid, username, len(records), duration, total)
		}
	}
	msg := fmt.Sprintf("loadUserRecord cnt:%d", cnt)
	util.SendCronMail(msg)
}
