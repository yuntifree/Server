package main

import (
	"log"
	"os"

	juhe "../juhe"
	util "../util"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	db, err := util.InitDB(false)
	if err != nil {
		log.Printf("InitDB failed:%v", err)
		os.Exit(1)
	}

	w, err := juhe.GetRealWeather()
	if err != nil {
		log.Printf("get real weather failed:%v", err)
		return
	}

	log.Printf("temperature:%d info:%s date:%s\n", w.Temperature, w.Info, w.Date)
	_, err = db.Exec("INSERT IGNORE INTO weather(temp, info, ctime, dtime) VALUES (?, ?, ?, NOW())", w.Temperature, w.Info, w.Date)
	if err != nil {
		log.Printf("insert into weather failed:%v", err)
	}
}
