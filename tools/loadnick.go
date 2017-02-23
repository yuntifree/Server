package main

import (
	"Server/util"
	"bufio"
	"io"
	"log"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	if len(os.Args) < 2 {
		log.Printf("not enough param")
		os.Exit(1)
	}
	filename := os.Args[1]
	db, err := util.InitDB(false)
	if err != nil {
		log.Printf("InitDB failed:%v", err)
		os.Exit(1)
	}
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	rd := bufio.NewReaderSize(f, 1024*1024)
	for {
		data, err := rd.ReadSlice('\n')
		if err != nil && err != io.EOF {
			log.Printf("failed:%v", err)
			break
		}
		line := string(data)
		arr := strings.Split(line, " ")
		log.Printf("arr len:%d", len(arr))
		for i := 0; i < len(arr); i++ {
			log.Printf("name:%s", arr[i])
			_, err = db.Exec("INSERT INTO nickname(name, ctime) VALUES (?, NOW())", arr[i])
			if err != nil {
				log.Printf("insert nickname failed:%s %v", arr[i], err)
			}
		}
	}
	return
}
