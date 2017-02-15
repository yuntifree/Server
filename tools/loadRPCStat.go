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
	requestType   = 1
	responseType  = 2
	statInterval  = 180
	nanoPerSecond = 1000000000
)

var db *sql.DB
var msgChan chan *nsq.Message

type rpcStat struct {
	Service  string
	Method   string
	ReqNum   int64
	SuccResp int64
}

type rpcMonitor struct {
	Start time.Time
	Mp    map[string]rpcStat
}

var monitor rpcMonitor

func flashStat(db *sql.DB, m *rpcMonitor) {
	for k, v := range m.Mp {
		log.Printf("service:%s method:%s req:%d succ resp:%d", v.Service,
			v.Method, v.ReqNum, v.SuccResp)
		_, err := db.Exec("INSERT INTO rpc_stat(service, method, req, succrsp, ctime) VALUES (?, ?, ?, ?, ?)",
			v.Service, v.Method, v.ReqNum, v.SuccResp,
			m.Start.Add(time.Second*statInterval).Format(util.TimeFormat))
		if err != nil {
			log.Printf("insert failed:%s %d %d %v", k, v.ReqNum, v.SuccResp, err)
		}
	}
	m.Mp = make(map[string]rpcStat)
}

func record(logChan chan *nsq.Message, m *rpcMonitor) {
	for {
		select {
		case msg := <-logChan:
			calc(msg, m)
		}
	}
}

func calc(msg *nsq.Message, m *rpcMonitor) {
	if msg.Timestamp < m.Start.UnixNano() {
		log.Printf("late msg to drop:%d %s", msg.Timestamp, string(msg.Body))
		return
	} else if msg.Timestamp >= m.Start.Add(time.Second*statInterval).UnixNano() {
		log.Printf("new msg to flash stat info :%d", msg.Timestamp)
		flashStat(db, m)
		m.Start = adjustStart(msg.Timestamp)
	}
	js, err := simplejson.NewJson(msg.Body)
	if err != nil {
		log.Printf("HandlerMessage parse body failed:%v", err)
		return
	}
	service, _ := js.Get("service").String()
	method, _ := js.Get("method").String()
	rtype, _ := js.Get("type").Int64()
	key := service + "." + method
	if stat, ok := m.Mp[key]; ok {
		if rtype == requestType {
			stat.ReqNum += 1
		} else if rtype == responseType {
			stat.SuccResp += 1
		}
		m.Mp[key] = stat
	} else {
		var stat rpcStat
		stat.Service = service
		stat.Method = method
		if rtype == requestType {
			stat.ReqNum += 1
		} else if rtype == responseType {
			stat.SuccResp += 1
		}
		m.Mp[key] = stat
	}

	return
}

func getPrevTime(tt time.Time) time.Time {
	year, month, day := tt.Date()
	local := tt.Location()
	hour, min, _ := tt.Clock()
	min = (min / 3) * 3
	return time.Date(year, month, day, hour, min, 0, 0, local)
}

func getNextTime(tt time.Time) time.Time {
	year, month, day := tt.Date()
	local := tt.Location()
	hour, min, _ := tt.Clock()
	min = (min/3 + 1) * 3
	return time.Date(year, month, day, hour, min, 0, 0, local)
}

func getStart() time.Time {
	now := time.Now()
	return getPrevTime(now)
}

func adjustStart(ts int64) time.Time {
	tm := time.Unix(ts/nanoPerSecond, ts%nanoPerSecond)
	return getPrevTime(tm)
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

	q, err := nsq.NewConsumer("rpc_monitor", "ch", config)
	if err != nil {
		log.Fatal(err)
	}
	monitor.Mp = make(map[string]rpcStat)
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
