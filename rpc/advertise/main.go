package main

import (
	"log"
	"net"
	"time"

	"golang.org/x/net/context"

	"Server/proto/advertise"
	"Server/proto/common"
	"Server/util"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	nsq "github.com/nsqio/go-nsq"
	"google.golang.org/grpc"
)

type server struct{}

var db *gorm.DB
var w *nsq.Producer

//Customer for customer info
type Customer struct {
	ID      int64  `gorm:"primary_key"`
	Name    string `gorm:"size:255"`
	Contact string `gorm:"size:255"`
	Phone   string `gorm:"size:16"`
	Address string `gorm:"size:255"`
	Remark  string `gorm:"size:255"`
	deleted int
	Ctime   time.Time
	Atime   string
	Etime   string
}

func addCustomer(db *gorm.DB, info *advertise.CustomerInfo) int64 {
	customer := Customer{Name: info.Name, Contact: info.Contact,
		Phone: info.Phone, Address: info.Address,
		Remark: info.Remark, Ctime: time.Now(),
		Atime: info.Atime, Etime: info.Etime,
	}
	db.Create(&customer)
	return customer.ID
}

func (s *server) AddCustomer(ctx context.Context, in *advertise.CustomerRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "advertise", "AddCustomer")
	id := addCustomer(db, in.Info)
	util.PubRPCSuccRsp(w, "advertise", "AddCustomer")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0}, Id: id}, nil
}

func main() {
	lis, err := net.Listen("tcp", util.AdvertiseServerPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	w = util.NewNsqProducer()
	db, err = util.InitOrm()
	if err != nil {
		log.Fatalf("failed to init db connection: %v", err)
	}
	db.DB().SetMaxIdleConns(util.MaxIdleConns)
	kv := util.InitRedis()
	go util.ReportHandler(kv, util.AdvertiseServerName, util.AdvertiseServerPort)
	//cli := util.InitEtcdCli()
	//go util.ReportEtcd(cli, util.AdvertiseServerName, util.AdvertiseServerPort)

	s := grpc.NewServer()
	advertise.RegisterAdvertiseServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
