package main

import (
	"Server/util"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

var apiList = []string{
	"one_click_login",
	"portal_login",
}

type rpcMethod struct {
	Service string
	Method  string
}

var rpcList = []rpcMethod{
	{"discover", "Resolve"},
	{"verify", "CheckLogin"},
	{"verify", "WifiAccess"},
	{"verify", "CheckToken"},
	{"verify", "OneClickLogin"},
	{"verify", "PortalLogin"},
	{"hot", "GetHots"},
	{"config", "GetPortalMenu"},
}

func checkApiStat(db *sql.DB, api string) {
	var req, succ int64
	var ctime string
	err := db.QueryRow("SELECT req, succrsp, ctime FROM api_stat WHERE name = ? ORDER BY id DESC LIMIT 1", api).Scan(&req, &succ, &ctime)
	if err != nil {
		log.Printf("checkApiStat query failed:%v", err)
		return
	}
	log.Printf("api:%s req:%d succrsp:%d ctime:%s", api, req, succ, ctime)

	if req >= 10 {
		rate := succ * 100 / req
		if rate < 95 {
			content := fmt.Sprintf("%s API:%s req:%d succ:%d rate:%d",
				ctime, api, req, succ, rate)
			log.Printf("SendAlertMail %s", content)
			err = util.SendAlertMail(content)
			if err != nil {
				log.Printf("SendAlertMail failed:%s %v", content, err)
			}
		}
	}
}

func checkRpcStat(db *sql.DB, rpc rpcMethod) {
	var req, succ int64
	var ctime string
	err := db.QueryRow("SELECT req, succrsp, ctime FROM rpc_stat WHERE service = ? AND method = ? ORDER BY id DESC LIMIT 1", rpc.Service, rpc.Method).Scan(&req, &succ, &ctime)
	if err != nil {
		log.Printf("checkApiStat query failed:%v", err)
		return
	}
	log.Printf("rpc:%s-%s req:%d succrsp:%d ctime:%s", rpc.Service, rpc.Method, req,
		succ, ctime)

	if req >= 10 {
		rate := succ * 100 / req
		if rate < 99 {
			content := fmt.Sprintf("%s RPC Method: %s-%s req:%d succ:%d rate:%d",
				ctime, rpc.Service, rpc.Method, req, succ, rate)
			log.Printf("SendAlertMail %s", content)
			err = util.SendAlertMail(content)
			if err != nil {
				log.Printf("SendAlertMail failed:%s %v", content, err)
			}
		}
	}
}

func main() {
	db, err := util.InitMonitorDB()
	if err != nil {
		log.Printf("InitDB failed:%v", err)
		os.Exit(1)
	}

	for _, api := range apiList {
		checkApiStat(db, api)
	}

	for _, rpc := range rpcList {
		checkRpcStat(db, rpc)
	}
}
