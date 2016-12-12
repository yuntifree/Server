package main

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"

	"../util"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	f, err := os.Open("address.txt")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	db, err := util.InitDB(false)
	if err != nil {
		panic(err)
	}

	rd := bufio.NewReader(f)
	for {
		data, err := rd.ReadSlice('\n')
		if err != nil {
			break
		}
		line := string(data)
		line = strings.TrimSpace(line)
		arr := strings.Split(line, " ")
		if len(arr) < 2 {
			continue
		}
		code, _ := strconv.Atoi(arr[0])
		address := arr[len(arr)-1]
		log.Printf("code:%d address:%s", code, address)
		if address == "市辖区" || address == "县" {
			continue
		}
		_, err = db.Exec("INSERT INTO zipcode(code, address) VALUES(?, ?)", code, address)
		if err != nil {
			log.Printf("insert zipcode failed:%d %s %v", code, address, err)
		}
	}
	db.Exec("INSERT INTO zipcode(code, address) VALUES(110100, '北京市辖区')")
	db.Exec("INSERT INTO zipcode(code, address) VALUES(110200, '北京市辖县')")
	db.Exec("INSERT INTO zipcode(code, address) VALUES(120100, '天津市辖区')")
	db.Exec("INSERT INTO zipcode(code, address) VALUES(120200, '天津市辖县')")
	db.Exec("INSERT INTO zipcode(code, address) VALUES(310100, '上海市辖区')")
	db.Exec("INSERT INTO zipcode(code, address) VALUES(310200, '上海市辖县')")
	db.Exec("INSERT INTO zipcode(code, address) VALUES(500100, '重庆市辖区')")
	db.Exec("INSERT INTO zipcode(code, address) VALUES(500200, '重庆市辖县')")

}
