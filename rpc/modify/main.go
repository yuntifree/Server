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
