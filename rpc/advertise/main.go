package main

import (
	"database/sql"
	"fmt"
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
	gdb := db.DB()
	res, err := gdb.Exec("INSERT INTO advertise(name, version, adid, areaid, tsid, abstract, img, content, dst, ctime) values (?, ?, ?, ?, ?, ?, ?, ?, ?, NOW())",
		ad.Name, ad.Version, ad.Adid, ad.Areaid, ad.Tsid, ad.Abstract,
		ad.Img, ad.Content, ad.Dst)
	id, err := res.LastInsertId()
	if err != nil {
		log.Printf("addAdvertise get id failed:%v", err)
		return 0
	}
	return id
}

func (s *server) AddAdvertise(ctx context.Context, in *advertise.AdvertiseRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "advertise", "AddAdvertise")
	id := addAdvertise(db, in.Info)
	util.PubRPCSuccRsp(w, "advertise", "AddAdvertise")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0}, Id: id}, nil
}

func modAdvertise(db *gorm.DB, in *advertise.AdvertiseRequest) {
	ad := locAdvertise(*in.Info)
	gdb := db.DB()
	gdb.Exec("UPDATE advertise SET name = ?, version = ?, adid = ?, areaid = ?, tsid=?, abstract=?, img=?, content=?, dst=?, deleted = ? WHERE id = ?",
		ad.Name, ad.Version, ad.Adid, ad.Areaid, ad.Tsid, ad.Abstract,
		ad.Img, ad.Content, ad.Dst, ad.Deleted, ad.ID)
}

func (s *server) ModAdvertise(ctx context.Context, in *advertise.AdvertiseRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "advertise", "ModAdvertise")
	modAdvertise(db, in)
	util.PubRPCSuccRsp(w, "advertise", "ModAdvertise")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0}}, nil
}

func fetchAdvertise(db *gorm.DB, seq, num int64) []*advertise.AdvertiseInfo {
	var infos []*advertise.AdvertiseInfo
	query := "SELECT a.id, a.name, a.version, a.adid, a.areaid, a.tsid, a.abstract, a.img, a.content, a.online, c.name, ar.name, ts.content, a.dst FROM advertise a, area ar, timeslot ts, customer c WHERE a.adid = c.id AND a.areaid = ar.id AND a.tsid = ts.id AND a.deleted = 0"
	if seq != 0 {
		query += fmt.Sprintf(" AND a.id < %d", seq)
	}
	query += fmt.Sprintf(" ORDER BY a.id DESC LIMIT %d", num)
	rows, err := db.Raw(query).Rows()
	if err != nil {
		log.Printf("fetchAdvertise query failed:%v", err)
		return infos
	}
	defer rows.Close()
	for rows.Next() {
		var info advertise.AdvertiseInfo
		err := rows.Scan(&info.ID, &info.Name, &info.Version, &info.Adid,
			&info.Areaid, &info.Tsid, &info.Abstract, &info.Img, &info.Content,
			&info.Online, &info.Adname, &info.Area, &info.Timeslot, &info.Dst)
		if err != nil {
			log.Printf("fetchAdvertise scan failed:%v", err)
			continue
		}
		infos = append(infos, &info)
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

func recordAdClick(db *gorm.DB, in *advertise.AdRequest) {
	db.Exec("INSERT INTO ad_click(uid, aid, usermac, userip, apmac, ctime) VALUES(?, ?, ?, ?, NOW())", in.Head.Uid, in.Aid, in.Usermac, in.Userip, in.Apmac)
	db.Exec("INSERT INTO ad_click_stat(aid, ctime, cnt) VALUES (?, CURDATE(), 1) ON DUPLICATE KEY UPDATE cnt = cnt + 1", in.Aid)
	db.Exec("UPDATE advertise SET click = click + 1 WHERE id = ?", in.Aid)
}

func (s *server) ClickAd(ctx context.Context, in *advertise.AdRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "advertise", "FetchCustomer")
	recordAdClick(db, in)
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

func fetchAreaUnit(db *sql.DB, seq, num int64) []*advertise.AreaUnitInfo {
	var infos []*advertise.AreaUnitInfo
	query := "SELECT au.id, a.id, a.name, u.name, u.id, u.longitude, u.latitude, u.cnt FROM area_unit au, area a, unit u WHERE au.aid = a.id AND au.unid = u.id AND au.deleted = 0 AND a.deleted = 0 AND u.deleted = 0"
	if seq != 0 {
		query += fmt.Sprintf(" AND au.id < %d ", seq)
	}
	query += fmt.Sprintf(" ORDER BY au.id DESC LIMIT %d", num)

	rows, err := db.Query(query)
	if err != nil {
		log.Printf("fetchAreaUnit query failed:%v", err)
		return infos
	}

	defer rows.Close()
	for rows.Next() {
		var info advertise.AreaUnitInfo
		err := rows.Scan(&info.Id, &info.Aid, &info.Areaname, &info.Unit,
			&info.Unid, &info.Longitude, &info.Latitude, &info.Cnt)
		if err != nil {
			log.Printf("fetchAreaUnit scan failed:%v", err)
			continue
		}
		infos = append(infos, &info)
	}
	return infos
}

func getTotalAreaUnit(db *sql.DB) int64 {
	var total int64
	err := db.QueryRow("SELECT COUNT(id) FROM area_unit WHERE deleted = 0").Scan(&total)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("getTotalAreaUnit query failed:%v", err)
	}
	return total
}

func (s *server) FetchAreaUnit(ctx context.Context, in *common.CommRequest) (*advertise.AreaUnitReply, error) {
	util.PubRPCRequest(w, "advertise", "FetchAreaUnit")
	gdb := db.DB()
	infos := fetchAreaUnit(gdb, in.Seq, in.Num)
	total := getTotalAreaUnit(gdb)
	util.PubRPCSuccRsp(w, "advertise", "FetchAreaUnit")
	return &advertise.AreaUnitReply{
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

func getTotalAdRecords(db *gorm.DB) int64 {
	var total int64
	gdb := db.DB()
	err := gdb.QueryRow("SELECT COUNT(id) FROM ad_click").Scan(&total)
	if err != nil {
		log.Printf("getTotalAdRecords query failed:%v", err)
	}
	return total
}

func fetchAdRecords(db *gorm.DB, seq, num int64) []*advertise.AdClickInfo {
	var infos []*advertise.AdClickInfo
	query := "SELECT c.id, c.uid, c.aid, c.usermac, c.userip, u.phone, a.name FROM ad_click c, advertise a, user u WHERE c.aid = a.id AND c.uid = u.uid"
	if seq != 0 {
		query += fmt.Sprintf(" AND c.id < %d ", seq)
	}
	query += fmt.Sprintf(" ORDER BY c.id DESC LIMIT %d", num)
	rows, err := db.Raw(query).Rows()
	if err != nil {
		log.Printf("query failed:%v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var info advertise.AdClickInfo
		err := rows.Scan(&info.Id, &info.Uid, &info.Aid, &info.Phone,
			&info.Usermac, &info.Userip, &info.Ctime)
		if err != nil {
			log.Printf("scan failed:%v", err)
			continue
		}
		infos = append(infos, &info)
	}
	return infos
}

func (s *server) FetchAdRecords(ctx context.Context, in *common.CommRequest) (*advertise.AdRecordsReply, error) {
	util.PubRPCRequest(w, "advertise", "FetchAdRecords")
	infos := fetchAdRecords(db, in.Seq, in.Num)
	total := getTotalAdRecords(db)
	util.PubRPCSuccRsp(w, "advertise", "FetchAdRecords")
	return &advertise.AdRecordsReply{
		Head: &common.Head{Retcode: 0}, Infos: infos, Total: total}, nil
}

func getParamInfo(db *gorm.DB, table string) []*advertise.ParamInfo {
	var infos []*advertise.ParamInfo
	rows, err := db.Table(table).Select("id, name").Rows()
	if err != nil {
		log.Printf("getParamInfo failed:%s %v", table, err)
		return infos
	}
	defer rows.Close()
	for rows.Next() {
		var info advertise.ParamInfo
		err := rows.Scan(&info.Id, &info.Name)
		if err != nil {
			log.Printf("getCustomerParam scan failed:%v", err)
		}
		infos = append(infos, &info)
	}
	return infos
}

func getCustomerParam(db *gorm.DB) []*advertise.ParamInfo {
	return getParamInfo(db, "customer")
}

func getAreaParam(db *gorm.DB) []*advertise.ParamInfo {
	return getParamInfo(db, "area")
}

func getTimeslotParam(db *gorm.DB) []*advertise.ParamInfo {
	return getParamInfo(db, "timeslot")
}

func (s *server) FetchAdParam(ctx context.Context, in *common.CommRequest) (*advertise.AdParamReply, error) {
	util.PubRPCRequest(w, "advertise", "FetchAdParam")
	customer := getCustomerParam(db)
	area := getAreaParam(db)
	timeslot := getTimeslotParam(db)
	util.PubRPCSuccRsp(w, "advertise", "FetchAdParam")
	return &advertise.AdParamReply{
		Head: &common.Head{Retcode: 0}, Customer: customer, Area: area,
		Timeslot: timeslot}, nil
}

func modAreaUnit(db *gorm.DB, mtype, aid int64, units []int64) {
	rdb := db.DB()
	for i := 0; i < len(units); i++ {
		if mtype == 0 {
			rdb.Exec("INSERT INTO area_unit(aid, unid, ctime) VALUES (?, ?, NOW()) ON DUPLICATE KEY UPDATE deleted = 0", aid, units[i])
		} else {
			rdb.Exec("UPDATE area_unit SET deleted = 1 WHERE aid = ? AND unid = ?",
				aid, units[i])
		}
	}
}

func (s *server) ModAreaUnit(ctx context.Context, in *advertise.AreaUnitRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "advertise", "ModAreaUnit")
	modAreaUnit(db, in.Type, in.Aid, in.Units)
	util.PubRPCSuccRsp(w, "advertise", "ModAreaUnit")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0},
	}, nil
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

	s := util.NewGrpcServer()
	advertise.RegisterAdvertiseServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
