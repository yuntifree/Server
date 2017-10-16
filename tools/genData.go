package main

import (
	"Server/util"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const (
	signNum    = 11763
	visitNum   = 14640
	dayNum     = 1000
	randNum    = 200
	daySeconds = 86400
	start      = 3600 * 8
	end        = 3600 * 21
)

type PhoneTime struct {
	Phone string
	Ts    int64
}

type ByTime []PhoneTime

func (t ByTime) Len() int           { return len(t) }
func (t ByTime) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }
func (t ByTime) Less(i, j int) bool { return t[i].Ts < t[j].Ts }

func main() {
	db, err := util.InitDB(false)
	if err != nil {
		log.Fatal(err)
	}
	rand.Seed(time.Now().Unix())
	//genSigninData(db)
	genVisitData(db)
}

func genVisitData(db *sql.DB) {
	phones := getPhones(db, visitNum)
	for i := 1; i <= 28; i++ {
		num := dayNum + randNum + rand.Int31()%randNum
		dayPhones := getRandPhones(phones, num)
		start := getStartTime(i)
		var bt []PhoneTime
		for j := 0; j < len(dayPhones); j++ {
			ts := genTs(start)
			var p PhoneTime
			p.Phone = dayPhones[j]
			p.Ts = ts
			bt = append(bt, p)
		}
		sort.Sort(ByTime(bt))
		for k := 0; k < len(bt); k++ {
			t := time.Unix(bt[k].Ts, 0)
			fmt.Printf("%s,%s\n", bt[k].Phone, t.Format(util.TimeFormat))
		}
	}
}

func genSigninData(db *sql.DB) {
	phones := getPhones(db, signNum)
	for i := 1; i <= 28; i++ {
		num := dayNum + rand.Int31()%randNum
		dayPhones := getRandPhones(phones, num)
		start := getStartTime(i)
		var bt []PhoneTime
		for j := 0; j < len(dayPhones); j++ {
			ts := genTs(start)
			var p PhoneTime
			p.Phone = dayPhones[j]
			p.Ts = ts
			bt = append(bt, p)
		}
		sort.Sort(ByTime(bt))
		for k := 0; k < len(bt); k++ {
			t := time.Unix(bt[k].Ts, 0)
			fmt.Printf("%s,%s\n", bt[k].Phone, t.Format(util.TimeFormat))
		}
	}
}

func genTs(start int64) int64 {
	r := rand.Int31() % 100
	var ts int32
	if r >= 80 { //0-8 || 21-24
		ts = rand.Int31() % (3600 * 10)
		if ts > 8*3600 {
			ts = ts + 13*3600
		}
	} else {
		ts = rand.Int31() % (3600 * 14)
		ts += 8 * 3600
	}
	return int64(ts) + start
}

func getStartTime(d int) int64 {
	t := time.Now()
	return time.Date(t.Year(), t.Month(), d, 0, 0, 0, 0, t.Location()).Unix()
}

func getRandPhones(phones []string, num int32) []string {
	if len(phones) < int(num) {
		return phones
	}
	n := rand.Int31() % (int32(len(phones)) - num)
	return phones[n : n+num]
}

func getPhones(db *sql.DB, num int) []string {
	var phones []string
	rows, err := db.Query(`SELECT phone FROM user WHERE phone != '' LIMIT ?`, num)
	if err != nil {
		return phones
	}

	defer rows.Close()
	for rows.Next() {
		var phone string
		err = rows.Scan(&phone)
		if err != nil {
			continue
		}
		phones = append(phones, phone)
	}
	return phones
}
