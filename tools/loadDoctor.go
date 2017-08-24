package main

import (
	"Server/util"
	"bufio"
	"log"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	db, err := util.InitInquiryDB()
	if err != nil {
		log.Printf("InitDB failed:%v", err)
		os.Exit(1)
	}

	f, err := os.Open("doctors.csv")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	rd := bufio.NewReader(f)
	for {
		data, err := rd.ReadSlice('\n')
		if err != nil {
			break
		}
		line := string(data)
		arr := strings.Split(line, ",")
		if len(arr) < 6 || arr[1] == "" {
			continue
		}
		res, err := db.Exec("INSERT IGNORE INTO doctor(name, phone, department, title, hospital, ctime) VALUES (?, ?, ?, ?, ?, NOW())",
			arr[0], arr[1], arr[2], arr[3], arr[4])
		if err != nil {
			log.Printf("insert failed:%s %v", line, err)
		}
		cnt, err := res.RowsAffected()
		if err != nil {
			log.Printf("get affected rows failed:%s %v", line, err)
		}
		if cnt == 0 {
			log.Printf("duplicated phone:%s", line)
		}
	}

}
