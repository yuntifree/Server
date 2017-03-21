package main

import (
	"Server/util"
	"database/sql"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func importOnlineRecord(db *sql.DB) {
	now := getStart()
	rows, err := db.Query("SELECT unid, cnt FROM (SELECT COUNT(o.id) AS cnt, a.unid FROM online_status o, ap_info a WHERE o.apmac = a.mac AND o.etime > NOW() GROUP BY a.unid) AS tl ORDER BY tl.cnt DESC")
	if err != nil {
		log.Printf("importOnlineRecord query failed:%v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var unid, cnt int
		err := rows.Scan(&unid, &cnt)
		if err != nil {
			log.Printf("importOnlineRecord scan failed:%v", err)
			continue
		}
		_, err = db.Exec("INSERT IGNORE INTO online_stat(unid, cnt, ctime) VALUES(?, ?, ?)", unid, cnt, now.Format(util.TimeFormat))
		if err != nil {
			log.Printf("INSERT failed:%v", err)
		}
	}
}

func getPrevTime(tt time.Time) time.Time {
	year, month, day := tt.Date()
	local := tt.Location()
	hour, min, _ := tt.Clock()
	min = (min / 5) * 5
	return time.Date(year, month, day, hour, min, 0, 0, local)
}

func getStart() time.Time {
	now := time.Now()
	return getPrevTime(now)
}

func main() {
	db, err := util.InitDB(false)
	if err != nil {
		log.Printf("InitDB failed:%v", err)
		os.Exit(1)
	}
	importOnlineRecord(db)

	return
}
