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

type locUnit advertise.UnitInfo

func (u locUnit) TableName() string {
	return "unit"
}

type locArea advertise.AreaInfo

func (a locArea) TableName() string {
	return "area"
}

type locTimeslot advertise.TimeslotInfo

func (a locTimeslot) TableName() string {
	return "timeslot"
}

func addUnit(db *gorm.DB, info *advertise.UnitInfo) int64 {
	u := locUnit(*info)
	u.Ctime = time.Now().Format(util.TimeFormat)
	db.Create(&u)
	return u.ID
}

func (s *server) AddUnit(ctx context.Context, in *advertise.UnitRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "advertise", "AddUnit")
	id := addUnit(db, in.Info)
	util.PubRPCSuccRsp(w, "advertise", "AddUnit")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0}, Id: id}, nil
}

func modUnit(db *gorm.DB, info *advertise.UnitInfo) {
	u := locUnit(*info)
	db.Save(&u)
}

func (s *server) ModUnit(ctx context.Context, in *advertise.UnitRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "advertise", "ModUnit")
	modUnit(db, in.Info)
	util.PubRPCSuccRsp(w, "advertise", "ModUnit")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0}}, nil
}

func fetchUnit(db *gorm.DB, seq, num int64) []*advertise.UnitInfo {
	var infos []*advertise.UnitInfo
	var ads []locUnit
	if seq == 0 {
		db.Where("deleted = 0").Order("id desc").Limit(num).Find(&ads)
	} else {
		db.Where("deleted = 0 AND id < ?", seq).Order("id desc").Limit(num).Find(&ads)
	}
	if len(ads) > 0 {
		for i := 0; i < len(ads); i++ {
			info := advertise.UnitInfo(ads[i])
			infos = append(infos, &info)
		}
	}
	return infos
}

func getTotalUnit(db *gorm.DB) int64 {
	var count int64
	db.Model(&locUnit{}).Where("deleted = 0").Count(&count)
	return count
}

func (s *server) FetchUnit(ctx context.Context, in *common.CommRequest) (*advertise.UnitReply, error) {
	util.PubRPCRequest(w, "advertise", "FetchUnit")
	infos := fetchUnit(db, in.Seq, in.Num)
	total := getTotalUnit(db)
	util.PubRPCSuccRsp(w, "advertise", "FetchUnit")
	return &advertise.UnitReply{
		Head: &common.Head{Retcode: 0}, Infos: infos, Total: total}, nil
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

func fetchAdvertise(db *gorm.DB, seq, num int64) []*advertise.AdvertiseInfo {
	var infos []*advertise.AdvertiseInfo
	var ads []locAdvertise
	if seq == 0 {
		db.Where("deleted = 0").Order("id desc").Limit(num).Find(&ads)
	} else {
		db.Where("deleted = 0 AND id < ?", seq).Order("id desc").Limit(num).Find(&ads)
	}
	if len(ads) > 0 {
		for i := 0; i < len(ads); i++ {
			info := advertise.AdvertiseInfo(ads[i])
			infos = append(infos, &info)
		}
	}
	return infos
}

func getTotalAdvertise(db *gorm.DB) int64 {
	var count int64
	db.Model(&locAdvertise{}).Where("deleted = 0").Count(&count)
	return count
}

func (s *server) FetchAdvertise(ctx context.Context, in *common.CommRequest) (*advertise.AdvertiseReply, error) {
	util.PubRPCRequest(w, "advertise", "FetchAdvertise")
	infos := fetchAdvertise(db, in.Seq, in.Num)
	total := getTotalAdvertise(db)
	util.PubRPCSuccRsp(w, "advertise", "FetchAdvertise")
	return &advertise.AdvertiseReply{
		Head: &common.Head{Retcode: 0}, Infos: infos, Total: total}, nil
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

func recordAdClick(db *gorm.DB, uid, id int64) {
	db.Exec("INSERT INTO ad_click(uid, aid, ctime) VALUES(?, ?, NOW())", uid, id)
	db.Exec("INSERT INTO ad_click_stat(aid, ctime, cnt) VALUES (?, CURDATE(), 1) ON DUPLICATE KEY UPDATE cnt = cnt + 1", id)
	db.Exec("UPDATE advertise SET click = click + 1 WHERE id = ?", id)
}

func (s *server) ClickAd(ctx context.Context, in *common.CommRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "advertise", "FetchCustomer")
	recordAdClick(db, in.Head.Uid, in.Id)
	util.PubRPCSuccRsp(w, "advertise", "FetchCustomer")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0}}, nil
}

func addArea(db *gorm.DB, info *advertise.AreaInfo) int64 {
	area := locArea(*info)
	area.Ctime = time.Now().Format(util.TimeFormat)
	db.Create(&area)
	return area.ID
}

func (s *server) AddArea(ctx context.Context, in *advertise.AreaRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "advertise", "AddArea")
	id := addArea(db, in.Info)
	util.PubRPCSuccRsp(w, "advertise", "AddArea")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0}, Id: id}, nil
}

func modArea(db *gorm.DB, info *advertise.AreaInfo) {
	in := locArea(*info)
	db.Save(&in)
}

func (s *server) ModArea(ctx context.Context, in *advertise.AreaRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "advertise", "ModArea")
	modArea(db, in.Info)
	util.PubRPCSuccRsp(w, "advertise", "ModArea")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0}}, nil
}

func fetchArea(db *gorm.DB, seq, num int64) []*advertise.AreaInfo {
	var infos []*advertise.AreaInfo
	var res []locArea
	if seq == 0 {
		db.Where("deleted = 0").Order("id desc").Limit(num).Find(&res)
	} else {
		db.Where("deleted = 0 AND id < ?", seq).Order("id desc").Limit(num).Find(&res)
	}
	if len(res) > 0 {
		for i := 0; i < len(res); i++ {
			info := advertise.AreaInfo(res[i])
			infos = append(infos, &info)
		}
	}
	return infos
}

func getTotalArea(db *gorm.DB) int64 {
	var count int64
	db.Model(&locArea{}).Where("deleted = 0").Count(&count)
	return count
}

func (s *server) FetchArea(ctx context.Context, in *common.CommRequest) (*advertise.AreaReply, error) {
	util.PubRPCRequest(w, "advertise", "FetchArea")
	infos := fetchArea(db, in.Seq, in.Num)
	total := getTotalArea(db)
	util.PubRPCSuccRsp(w, "advertise", "FetchArea")
	return &advertise.AreaReply{
		Head: &common.Head{Retcode: 0}, Infos: infos, Total: total}, nil
}

func addTimeslot(db *gorm.DB, info *advertise.TimeslotInfo) int64 {
	in := locTimeslot(*info)
	in.Ctime = time.Now().Format(util.TimeFormat)
	db.Create(&in)
	return in.ID
}

func (s *server) AddTimeslot(ctx context.Context, in *advertise.TimeslotRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "advertise", "AddTimeslot")
	id := addTimeslot(db, in.Info)
	util.PubRPCSuccRsp(w, "advertise", "AddTimeslot")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0}, Id: id}, nil
}

func modTimeslot(db *gorm.DB, info *advertise.TimeslotInfo) {
	in := locTimeslot(*info)
	db.Save(&in)
}

func (s *server) ModTimeslot(ctx context.Context, in *advertise.TimeslotRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "advertise", "ModTimeslot")
	modTimeslot(db, in.Info)
	util.PubRPCSuccRsp(w, "advertise", "ModTimeslot")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0}}, nil
}

func fetchTimeslot(db *gorm.DB, seq, num int64) []*advertise.TimeslotInfo {
	var infos []*advertise.TimeslotInfo
	var res []locTimeslot
	if seq == 0 {
		db.Where("deleted = 0").Order("id desc").Limit(num).Find(&res)
	} else {
		db.Where("deleted = 0 AND id < ?", seq).Order("id desc").Limit(num).Find(&res)
	}
	if len(res) > 0 {
		for i := 0; i < len(res); i++ {
			info := advertise.TimeslotInfo(res[i])
			infos = append(infos, &info)
		}
	}
	return infos
}

func getTotalTimeslot(db *gorm.DB) int64 {
	var count int64
	db.Model(&locTimeslot{}).Where("deleted = 0").Count(&count)
	return count
}

func (s *server) FetchTimeslot(ctx context.Context, in *common.CommRequest) (*advertise.TimeslotReply, error) {
	util.PubRPCRequest(w, "advertise", "FetchTimeslot")
	infos := fetchTimeslot(db, in.Seq, in.Num)
	total := getTotalTimeslot(db)
	util.PubRPCSuccRsp(w, "advertise", "FetchTimeslot")
	return &advertise.TimeslotReply{
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
