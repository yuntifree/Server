package main

import (
	"log"
	"os"
	"time"

	util "../util"
	zte "../zte"
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
	db, err := util.InitDB(false)
	if err != nil {
		log.Printf("InitDB failed:%v", err)
		os.Exit(1)
	}
	rows, err := db.Query("SELECT uid, username, CURDATE(), DATE_SUB(CURDATE(), INTERVAL 1 DAY) FROM user WHERE aptime > DATE_SUB(CURDATE(), INTERVAL 1 DAY)")
	if err != nil {
		log.Fatalf("query failed:%v", err)
	}

	defer rows.Close()
	for rows.Next() {
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
			for i := 0; i < len(records); i++ {
				_, err := db.Exec("INSERT INTO user_record(uid, aid, stime, etime) VALUES (?, ?, ?, ?)", uid, records[i].Aid, records[i].Start, records[i].End)
				if err != nil {
					log.Printf("insert failed:%v", err)
					continue
				}
				duration += diff(records[i].End, records[i].Start)
			}
			_, err := db.Exec("UPDATE user SET times = times + ?, duration = duration + ? WHERE uid = ?", len(records), duration, uid)
			if err != nil {
				log.Printf("update user info failed:%v", err)
			}
			log.Printf("uid:%d username:%s times:%d duration:%d\n", uid, username, len(records), duration)
		}
	}
}
