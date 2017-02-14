package main

import (
	"Server/util"
	"database/sql"
	"log"
	"net"
	"os"
	"time"

	simplejson "github.com/bitly/go-simplejson"
	_ "github.com/go-sql-driver/mysql"
	nsq "github.com/nsqio/go-nsq"
)

const (
	requestType  = 1
	responseType = 2
	statInterval = 180
)

var db *sql.DB
var msgChan chan *nsq.Message

type apiStat struct {
	ReqNum   int64
	SuccResp int64
}

type apiMonitor struct {
	Start time.Time
	Mp    map[string]apiStat
}

var monitor apiMonitor

func flashStat(db *sql.DB, m *apiMonitor) {
	for k, v := range m.Mp {
		log.Printf("api:%s req:%d succ resp:%d", k, v.ReqNum, v.SuccResp)
		_, err := db.Exec("INSERT INTO api_stat(name, req, succrsp, ctime) VALUES (?, ?, ?, ?)",
			k, v.ReqNum, v.SuccResp,
			m.Start.Add(time.Second*statInterval).Format(util.TimeFormat))
		if err != nil {
			log.Printf("insert failed:%s %d %d %v", k, v.ReqNum, v.SuccResp, err)
		}
	}
	m.Mp = make(map[string]apiStat)
}

func record(logChan chan *nsq.Message, m *apiMonitor) {
	for {
		select {
		case msg := <-logChan:
			calc(msg, m)
		}
	}
}

func calc(msg *nsq.Message, m *apiMonitor) {
	if msg.Timestamp < m.Start.UnixNano() {
		log.Printf("late msg to drop:%d %s", msg.Timestamp, string(msg.Body))
		return
	} else if msg.Timestamp >= m.Start.Add(time.Second*statInterval).UnixNano() {
		log.Printf("new msg to flash stat info :%d", msg.Timestamp)
		flashStat(db, m)
		m.Start = m.Start.Add(time.Second * statInterval)
	}
	js, err := simplejson.NewJson(msg.Body)
	if err != nil {
		log.Printf("HandlerMessage parse body failed:%v", err)
		return
	}
	api, _ := js.Get("name").String()
	rtype, _ := js.Get("type").Int64()
	if stat, ok := m.Mp[api]; ok {
		if rtype == requestType {
			stat.ReqNum += 1
		} else if rtype == responseType {
			stat.SuccResp += 1
		}
		m.Mp[api] = stat
	} else {
		var stat apiStat
		if rtype == requestType {
			stat.ReqNum += 1
		} else if rtype == responseType {
			stat.SuccResp += 1
		}
		m.Mp[api] = stat
	}

	return
}

func getStart() time.Time {
	now := time.Now()
	year, month, day := now.Date()
	local := now.Location()
	hour, min, _ := now.Clock()
	min = (min / 3) * 3
	return time.Date(year, month, day, hour, min, 0, 0, local)
}

func main() {
	done := make(chan bool)
	var err error
	db, err = util.InitMonitorDB()
	if err != nil {
		log.Printf("InitDB failed:%v", err)
		os.Exit(1)
	}
	config := nsq.NewConfig()
	laddr := "10.26.210.175"
	config.LocalAddr, _ = net.ResolveTCPAddr("tcp", laddr+":0")
	config.DefaultRequeueDelay = 0
	config.MaxBackoffDuration = time.Millisecond * 50

	q, err := nsq.NewConsumer("api_monitor", "ch", config)
	if err != nil {
		log.Fatal(err)
	}
	monitor.Mp = make(map[string]apiStat)
	monitor.Start = getStart()
	logChan := make(chan *nsq.Message, 100)
	q.AddHandler(nsq.HandlerFunc(func(m *nsq.Message) error {
		log.Printf("msg:%s", string(m.Body))
		logChan <- m
		return nil
	}))

	err = q.ConnectToNSQLookupd("10.26.210.175:4161")
	if err != nil {
		log.Printf("connect failed:%v", err)
	}
	go record(logChan, &monitor)
	<-done
	return
}
