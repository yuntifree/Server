package main

import (
	"Server/util"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

const (
	start = "2017-09-15"
)

func main() {
	db, err := util.InitDB(false)
	if err != nil {
		log.Printf("InitDB failed:%v", err)
		os.Exit(1)
	}

	for i := 0; i < 30; i++ {
		rows, err := db.Query(`SELECT u.phone, u.model, o.ctime FROM user u,
		online_record_201710 o WHERE u.uid = o.uid AND 
		(o.ctime > DATE_ADD(?, INTERVAL ? DAY) AND 
		o.ctime < DATE_ADD(DATE_ADD(?, INTERVAL ? DAY), INTERVAL 4 HOUR))`,
			start, i, start, i)
		if err != nil {
			log.Printf("query failed:%v", err)
			continue
		}
		defer rows.Close()
		for rows.Next() {
			var phone, model, ctime string
			err = rows.Scan(&phone, &model, &ctime)
			if err != nil {
				continue
			}
			fmt.Printf("%s,%s,%s\n", phone, model, ctime)
		}

		rows2, err := db.Query(`SELECT u.phone, u.model, o.ctime FROM user u,
		online_record_201710 o WHERE u.uid = o.uid AND 
		(o.ctime > DATE_ADD(DATE_ADD(?, INTERVAL ? DAY), INTERVAL 23 HOUR) AND 
		o.ctime < DATE_ADD(DATE_ADD(?, INTERVAL ? DAY), INTERVAL 24 HOUR))`,
			start, i, start, i)
		if err != nil {
			log.Printf("query failed:%v", err)
			continue
		}
		defer rows2.Close()
		for rows2.Next() {
			var phone, model, ctime string
			err = rows2.Scan(&phone, &model, &ctime)
			if err != nil {
				continue
			}
			fmt.Printf("%s,%s,%s\n", phone, model, ctime)
		}
	}
}
