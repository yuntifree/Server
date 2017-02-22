package main

import (
	"fmt"
	"log"
	"net"

	"database/sql"

	"Server/proto/common"
	"Server/proto/config"
	"Server/util"

	_ "github.com/go-sql-driver/mysql"
	nsq "github.com/nsqio/go-nsq"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	redis "gopkg.in/redis.v5"
)

const (
	menuType = 0
	tabType  = 1
)

type server struct{}

var db *sql.DB
var kv *redis.Client
var w *nsq.Producer

func getPortalMenu(db *sql.DB, stype int64, flag bool) []*config.PortalMenuInfo {
	query := fmt.Sprintf("SELECT icon, text, name, routername, url FROM portal_menu WHERE type = %d ", stype)
	if !flag {
		query += " AND dbg = 0 "
	}
	query += " ORDER BY priority DESC"
	rows, err := db.Query(query)
	var infos []*config.PortalMenuInfo
	if err != nil {
		log.Printf("getPortalMenu query failed:%v", err)
		return infos
	}

	defer rows.Close()
	for rows.Next() {
		var info config.PortalMenuInfo
		err := rows.Scan(&info.Icon, &info.Text, &info.Name, &info.Routername,
			&info.Url)
		if err != nil {
			log.Printf("getPortalMenu scan failed:%v", err)
			continue
		}
		infos = append(infos, &info)
	}
	return infos
}

func (s *server) GetPortalMenu(ctx context.Context, in *common.CommRequest) (*config.PortalMenuReply, error) {
	util.PubRPCRequest(w, "config", "GetPortalMenu")
	flag := util.IsWhiteUser(db, in.Head.Uid, util.PortalMenuDbgType)
	menulist := getPortalMenu(db, menuType, flag)
	tablist := getPortalMenu(db, tabType, flag)
	util.PubRPCSuccRsp(w, "config", "GetPortalMenu")
	return &config.PortalMenuReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Menulist: menulist,
		Tablist: tablist}, nil
}

func fetchPortalMenu(db *sql.DB, stype int64) []*config.PortalMenuInfo {
	var infos []*config.PortalMenuInfo
	rows, err := db.Query("SELECT id, icon, text, name, routername, url, priority, dbg, deleted FROM portal_menu WHERE type = ?", stype)
	if err != nil {
		log.Printf("fetchPortalMenu query failed:%v", err)
		return infos
	}

	defer rows.Close()
	for rows.Next() {
		var info config.PortalMenuInfo
		err := rows.Scan(&info.Id, &info.Icon, &info.Text, &info.Name, &info.Routername,
			&info.Url, &info.Priority, &info.Dbg, &info.Deleted)
		if err != nil {
			log.Printf("fetchPortalMenu scan failed:%v", err)
			continue
		}
		infos = append(infos, &info)
	}
	return infos
}

func (s *server) FetchPortalMenu(ctx context.Context, in *common.CommRequest) (*config.MenuReply, error) {
	util.PubRPCRequest(w, "config", "FetchPortalMenu")
	infos := fetchPortalMenu(db, in.Type)
	util.PubRPCSuccRsp(w, "config", "FetchPortalMenu")
	return &config.MenuReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Infos: infos}, nil
}

func (s *server) ModPortalMenu(ctx context.Context, in *config.MenuRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "config", "ModPortalMenu")
	query := fmt.Sprintf("UPDATE portal_menu SET mtime = NOW(), dbg = %d, deleted = %d ",
		in.Info.Dbg, in.Info.Deleted)
	if in.Info.Icon != "" {
		query += ", icon = '" + in.Info.Icon + "' "
	}
	if in.Info.Text != "" {
		query += ", text = '" + in.Info.Text + "' "
	}
	if in.Info.Name != "" {
		query += ", name = '" + in.Info.Name + "' "
	}
	if in.Info.Url != "" {
		query += ", url = '" + in.Info.Url + "' "
	}
	if in.Info.Priority != 0 {
		query += fmt.Sprintf(", priority = %d", in.Info.Priority)
	}
	query += fmt.Sprintf(" WHERE id = %d", in.Info.Id)
	_, err := db.Exec(query)
	if err != nil {
		log.Printf("ModPortalMenu query failed:%v", err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	util.PubRPCSuccRsp(w, "config", "ModPortalMenu")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func main() {
	lis, err := net.Listen("tcp", util.ConfigServerPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	w = util.NewNsqProducer()

	db, err = util.InitDB(false)
	if err != nil {
		log.Fatalf("failed to init db connection: %v", err)
	}
	db.SetMaxIdleConns(util.MaxIdleConns)
	kv = util.InitRedis()
	go util.ReportHandler(kv, util.ConfigServerName, util.ConfigServerPort)
	//cli := util.InitEtcdCli()
	//go util.ReportEtcd(cli, util.ConfigServerName, util.ConfigServerPort)

	s := grpc.NewServer()
	config.RegisterConfigServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
