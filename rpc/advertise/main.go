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

type locCustomer advertise.CustomerInfo

func (c locCustomer) TableName() string {
	return "customer"
}

type locAdvertise advertise.AdvertiseInfo

func (ad locAdvertise) TableName() string {
	return "advertise"
}

func addAdvertise(db *gorm.DB, info *advertise.AdvertiseInfo) int64 {
	ad := locAdvertise(*info)
	ad.Ctime = time.Now().Format(util.TimeFormat)
	db.Create(&ad)
	return ad.ID
}

func (s *server) AddAdvertise(ctx context.Context, in *advertise.AdvertiseRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "advertise", "AddAdvertise")
	id := addAdvertise(db, in.Info)
	util.PubRPCSuccRsp(w, "advertise", "AddAdvertise")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0}, Id: id}, nil
}

func modAdvertise(db *gorm.DB, info *advertise.AdvertiseInfo) {
	ad := locAdvertise(*info)
	db.Save(&ad)
}

func (s *server) ModAdvertise(ctx context.Context, in *advertise.AdvertiseRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "advertise", "ModAdvertise")
	modAdvertise(db, in.Info)
	util.PubRPCSuccRsp(w, "advertise", "ModAdvertise")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0}}, nil
}

func addCustomer(db *gorm.DB, info *advertise.CustomerInfo) int64 {
	customer := locCustomer(*info)
	customer.Ctime = time.Now().Format(util.TimeFormat)
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

func modCustomer(db *gorm.DB, info *advertise.CustomerInfo) {
	customer := locCustomer(*info)
	db.Save(&customer)
}

func (s *server) ModCustomer(ctx context.Context, in *advertise.CustomerRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "advertise", "ModCustomer")
	modCustomer(db, in.Info)
	util.PubRPCSuccRsp(w, "advertise", "ModCustomer")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0}}, nil
}

func fetchCustomer(db *gorm.DB, seq, num int64) []*advertise.CustomerInfo {
	var infos []*advertise.CustomerInfo
	var customers []locCustomer
	if seq == 0 {
		db.Where("deleted = 0").Order("id desc").Limit(num).Find(&customers)
	} else {
		db.Where("deleted = 0 AND id < ?", seq).Order("id desc").Limit(num).Find(&customers)
	}
	if len(customers) > 0 {
		for i := 0; i < len(customers); i++ {
			info := advertise.CustomerInfo(customers[i])
			infos = append(infos, &info)
		}
	}
	return infos
}

func getTotalCustomer(db *gorm.DB) int64 {
	var count int64
	db.Model(&locCustomer{}).Where("deleted = 0").Count(&count)
	return count
}

func (s *server) FetchCustomer(ctx context.Context, in *common.CommRequest) (*advertise.CustomerReply, error) {
	util.PubRPCRequest(w, "advertise", "FetchCustomer")
	infos := fetchCustomer(db, in.Seq, in.Num)
	total := getTotalCustomer(db)
	util.PubRPCSuccRsp(w, "advertise", "FetchCustomer")
	return &advertise.CustomerReply{
		Head: &common.Head{Retcode: 0}, Infos: infos, Total: total}, nil
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
