package main

import (
	"log"
	"net"
	"strconv"

	"../../util"

	common "../../proto/common"
	modify "../../proto/modify"
	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	port       = ":50056"
	servername = "service:modify"
)

type server struct{}

func (s *server) ReviewNews(ctx context.Context, in *modify.NewsRequest) (*modify.NewsReply, error) {
	db, err := util.InitDB(false)
	if err != nil {
		log.Printf("connect mysql failed:%v", err)
		return &modify.NewsReply{Head: &common.Head{Retcode: 1}}, err
	}
	defer db.Close()

	if in.Reject {
		db.Exec("UPDATE news SET review = 1, deleted = 1, rtime = NOW(), ruid = ? WHERE id = ?", in.Head.Uid, in.Id)
	} else {
		query := "UPDATE news SET review = 1, rtime = NOW(), ruid = " + strconv.Itoa(int(in.Head.Uid))
		if in.Modify && in.Title != "" {
			query += ", title = '" + in.Title + "' "
		}
		query += " WHERE id = " + strconv.Itoa(int(in.Id))
		db.Exec(query)
		if len(in.Tags) > 0 {
			for i := 0; i < len(in.Tags); i++ {
				db.Exec("INSERT INTO news_tags(nid, tid, ruid, ctime) VALUES (?, ?, ?, NOW())", in.Id, in.Tags[i], in.Head.Uid)
			}
		}
	}

	return &modify.NewsReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func (s *server) AddTemplate(ctx context.Context, in *modify.AddTempRequest) (*modify.AddTempReply, error) {
	db, err := util.InitDB(false)
	if err != nil {
		log.Printf("connect mysql failed:%v", err)
		return &modify.AddTempReply{Head: &common.Head{Retcode: 1}}, err
	}
	defer db.Close()

	res, err := db.Exec("INSERT INTO template(title, content, ruid, ctime, mtime) VALUES (?, ?, ?, NOW(), NOW())", in.Info.Title, in.Info.Content, in.Head.Uid)
	if err != nil {
		log.Printf("query failed:%v", err)
		return &modify.AddTempReply{Head: &common.Head{Retcode: 1}}, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		log.Printf("query failed:%v", err)
		return &modify.AddTempReply{Head: &common.Head{Retcode: 1}}, err
	}

	return &modify.AddTempReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Id: int32(id)}, nil
}

func (s *server) AddWifi(ctx context.Context, in *modify.WifiRequest) (*modify.WifiReply, error) {
	db, err := util.InitDB(false)
	if err != nil {
		log.Printf("connect mysql failed:%v", err)
		return &modify.WifiReply{Head: &common.Head{Retcode: 1}}, err
	}
	defer db.Close()

	_, err = db.Exec("INSERT INTO wifi(ssid, username, password, longitude, latitude, uid, ctime) VALUES (?, ?, ?, ?,?,?, NOW())", in.Info.Ssid, in.Info.Username, in.Info.Password, in.Info.Longitude, in.Info.Latitude, in.Head.Uid)
	if err != nil {
		log.Printf("query failed:%v", err)
		return &modify.WifiReply{Head: &common.Head{Retcode: 1}}, err
	}

	return &modify.WifiReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func (s *server) ModTemplate(ctx context.Context, in *modify.ModTempRequest) (*modify.ModTempReply, error) {
	db, err := util.InitDB(false)
	if err != nil {
		log.Printf("connect mysql failed:%v", err)
		return &modify.ModTempReply{Head: &common.Head{Retcode: 1}}, err
	}
	defer db.Close()

	query := "UPDATE template SET "
	if in.Info.Title != "" {
		query += " title = '" + in.Info.Title + "', "
	}
	if in.Info.Content != "" {
		query += " content = '" + in.Info.Content + "', "
	}
	online := 0
	if in.Info.Online {
		online = 1
	}
	query += " mtime = NOW(), ruid = " + strconv.Itoa(int(in.Head.Uid)) + ", online = " + strconv.Itoa(online) + " WHERE id = " + strconv.Itoa(int(in.Info.Id))
	_, err = db.Exec(query)

	if err != nil {
		log.Printf("query failed:%v", err)
		return &modify.ModTempReply{Head: &common.Head{Retcode: 1}}, err
	}

	return &modify.ModTempReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func (s *server) ReportClick(ctx context.Context, in *modify.ClickRequest) (*modify.ClickReply, error) {
	db, err := util.InitDB(false)
	if err != nil {
		log.Printf("connect mysql failed:%v", err)
		return &modify.ClickReply{Head: &common.Head{Retcode: 1}}, err
	}
	defer db.Close()

	res, err := db.Exec("INSERT IGNORE INTO click_record(uid, type, id, ctime) VALUES(?, ?, ?, NOW())", in.Head.Uid, in.Type, in.Id)
	if err != nil {
		log.Printf("query failed:%v", err)
		return &modify.ClickReply{Head: &common.Head{Retcode: 1}}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Printf("get last insert id failed:%v", err)
		return &modify.ClickReply{Head: &common.Head{Retcode: 1}}, err
	}

	if id != 0 {
		switch in.Type {
		case 0:
			_, err = db.Query("UPDATE youku_video SET play = play + 1 WHERE vid = ?", in.Id)
		case 1:
			_, err = db.Query("UPDATE news SET click = click + 1 WHERE id = ?", in.Id)
		case 2:
			_, err = db.Query("UPDATE ads SET display = display + 1 WHERE id = ?", in.Id)
		case 3:
			_, err = db.Query("UPDATE ads SET click = click + 1 WHERE id = ?", in.Id)
		default:
			log.Printf("illegal type:%d, id:%d uid:%d", in.Type, in.Id, in.Head.Uid)

		}
		if err != nil {
			log.Printf("update click count failed type:%d id:%d:%v", in.Type, in.Id, err)
			return &modify.ClickReply{Head: &common.Head{Retcode: 1}}, err
		}
	}

	return &modify.ClickReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func main() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	go util.ReportHandler(servername, port)

	s := grpc.NewServer()
	modify.RegisterModifyServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
