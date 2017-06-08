package main

import (
	"Server/proto/inquiry"
	"Server/util"
	"database/sql"
	"log"
	"net"

	_ "github.com/go-sql-driver/mysql"
	nsq "github.com/nsqio/go-nsq"
	redis "gopkg.in/redis.v5"
)

type server struct{}

var db *sql.DB
var kv *redis.Client
var w *nsq.Producer

func main() {
	lis, err := net.Listen("tcp", util.InquiryServerPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	db, err = util.InitInquiryDB()
	if err != nil {
		log.Fatalf("failed to init db connection: %v", err)
	}
	db.SetMaxIdleConns(util.MaxIdleConns)
	w = util.NewNsqProducer()

	kv = util.InitRedis()
	go util.ReportHandler(kv, util.InquiryServerName, util.InquiryServerPort)

	s := util.NewGrpcServer()
	inquiry.RegisterInquiryServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
