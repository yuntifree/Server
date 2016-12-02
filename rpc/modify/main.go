package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"

	"../../util"

	common "../../proto/common"
	modify "../../proto/modify"
	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type server struct{}

var db *sql.DB

func (s *server) ReviewNews(ctx context.Context, in *modify.NewsRequest) (*modify.CommReply, error) {
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

	return &modify.CommReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func (s *server) ReviewVideo(ctx context.Context, in *modify.VideoRequest) (*modify.CommReply, error) {
	if in.Reject {
		db.Exec("UPDATE youku_video SET review = 1, deleted = 1, rtime = NOW(), ruid = ? WHERE vid = ?", in.Head.Uid, in.Id)
	} else {
		query := "UPDATE youku_video SET review = 1, rtime = NOW(), ruid = " + strconv.Itoa(int(in.Head.Uid))
		if in.Modify && in.Title != "" {
			query += ", title = '" + in.Title + "' "
		}
		query += " WHERE vid = " + strconv.Itoa(int(in.Id))
		db.Exec(query)
	}

	return &modify.CommReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func (s *server) AddTemplate(ctx context.Context, in *modify.AddTempRequest) (*modify.AddTempReply, error) {
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

func (s *server) AddWifi(ctx context.Context, in *modify.WifiRequest) (*modify.CommReply, error) {
	_, err := db.Exec("INSERT INTO wifi(ssid, username, password, longitude, latitude, uid, ctime) VALUES (?, ?, ?, ?,?,?, NOW())", in.Info.Ssid, in.Info.Username, in.Info.Password, in.Info.Longitude, in.Info.Latitude, in.Head.Uid)
	if err != nil {
		log.Printf("query failed:%v", err)
		return &modify.CommReply{Head: &common.Head{Retcode: 1}}, err
	}

	return &modify.CommReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func (s *server) ModTemplate(ctx context.Context, in *modify.ModTempRequest) (*modify.CommReply, error) {
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
	_, err := db.Exec(query)

	if err != nil {
		log.Printf("query failed:%v", err)
		return &modify.CommReply{Head: &common.Head{Retcode: 1}}, err
	}

	return &modify.CommReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func (s *server) ReportClick(ctx context.Context, in *modify.ClickRequest) (*modify.CommReply, error) {
	var res sql.Result
	var err error
	if in.Type != 4 {
		res, err = db.Exec("INSERT IGNORE INTO click_record(uid, type, id, ctime) VALUES(?, ?, ?, NOW())", in.Head.Uid, in.Type, in.Id)
	} else {
		res, err = db.Exec("INSERT INTO service_click_record(uid, sid, ctime) VALUES(?, ?, NOW())", in.Head.Uid, in.Id)
	}
	if err != nil {
		log.Printf("query failed:%v", err)
		return &modify.CommReply{Head: &common.Head{Retcode: 1}}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Printf("get last insert id failed:%v", err)
		return &modify.CommReply{Head: &common.Head{Retcode: 1}}, err
	}

	if id != 0 {
		switch in.Type {
		case 0:
			_, err = db.Exec("UPDATE youku_video SET play = play + 1 WHERE vid = ?", in.Id)
		case 1:
			_, err = db.Exec("UPDATE news SET click = click + 1 WHERE id = ?", in.Id)
		case 2:
			_, err = db.Exec("UPDATE ads SET display = display + 1 WHERE id = ?", in.Id)
		case 3:
			_, err = db.Exec("UPDATE ads SET click = click + 1 WHERE id = ?", in.Id)
		case 4:
			_, err = db.Exec("INSERT INTO service_click(sid, click, ctime) VALUES (?, 1, CURDATE()) ON DUPLICATE KEY UPDATE click = click + 1", in.Id)
		default:
			log.Printf("illegal type:%d, id:%d uid:%d", in.Type, in.Id, in.Head.Uid)

		}
		if err != nil {
			log.Printf("update click count failed type:%d id:%d:%v", in.Type, in.Id, err)
			return &modify.CommReply{Head: &common.Head{Retcode: 1}}, err
		}
	}

	return &modify.CommReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func (s *server) ReportApmac(ctx context.Context, in *modify.ApmacRequest) (*modify.CommReply, error) {
	var aid int
	mac := strings.Replace(strings.ToLower(in.Apmac), ":", "", -1)
	log.Printf("ap mac origin:%s convert:%s\n", in.Apmac, mac)
	err := db.QueryRow("SELECT id FROM ap WHERE mac = ? OR mac = ?", in.Apmac, mac).Scan(&aid)
	if err != nil {
		log.Printf("select aid from ap failed uid:%d mac:%s err:%v\n", in.Head.Uid, in.Apmac, err)
		return &modify.CommReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
	}
	_, err = db.Exec("UPDATE user SET aid = ?, aptime = NOW() WHERE uid = ?", aid, in.Head.Uid)
	if err != nil {
		log.Printf("update user ap info failed uid:%d aid:%d\n", in.Head.Uid, aid)
	}
	return &modify.CommReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func (s *server) AddImage(ctx context.Context, in *modify.AddImageRequest) (*modify.CommReply, error) {
	for i := 0; i < len(in.Fnames); i++ {
		_, err := db.Exec("INSERT IGNORE INTO image(uid, name, ctime) VALUES(?, ?, NOW())",
			in.Head.Uid, in.Fnames[i])
		if err != nil {
			log.Printf("insert into image failed uid:%d name:%s err:%v\n", in.Head.Uid, in.Fnames[i], err)
		}
	}
	return &modify.CommReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func (s *server) FinImage(ctx context.Context, in *modify.ImageRequest) (*modify.CommReply, error) {
	_, err := db.Exec("UPDATE image SET filesize = ?, height = ?, width = ?, ftime = NOW(), status = 1 WHERE name = ?",
		in.Info.Size, in.Info.Height, in.Info.Width, in.Info.Name)
	if err != nil {
		log.Printf("update image failed name:%s err:%v\n", in.Info.Name, err)
	}
	return &modify.CommReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func (s *server) AddBanner(ctx context.Context, in *modify.BannerRequest) (*modify.CommReply, error) {
	res, err := db.Exec("INSERT INTO banner(img, dst, priority, ctime) VALUES(?, ?, ?, NOW())",
		in.Info.Img, in.Info.Dst, in.Info.Priority)
	if err != nil {
		log.Printf("insert into banner failed img:%s dst:%s err:%v\n", in.Info.Img, in.Info.Dst, err)
		return &modify.CommReply{Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Printf("AddBanner get LastInsertId failed:%v\n", err)
		return &modify.CommReply{Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	return &modify.CommReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Id: id}, nil
}

func (s *server) ModBanner(ctx context.Context, in *modify.BannerRequest) (*modify.CommReply, error) {
	query := fmt.Sprintf("UPDATE banner SET priority = %d, online = %d, deleted = %d ",
		in.Info.Priority, in.Info.Online, in.Info.Deleted)
	if in.Info.Img != "" {
		query += ", img = '" + in.Info.Img + "' "
	}
	if in.Info.Dst != "" {
		query += ", dst = '" + in.Info.Dst + "' "
	}
	query += fmt.Sprintf(" WHERE id = %d", in.Info.Id)
	_, err := db.Exec(query)
	if err != nil {
		log.Printf("insert into banner failed img:%s dst:%s err:%v\n", in.Info.Img, in.Info.Dst, err)
		return &modify.CommReply{Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	return &modify.CommReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func (s *server) AddTags(ctx context.Context, in *modify.AddTagRequest) (*modify.AddTagReply, error) {
	var ids []int32
	for i := 0; i < len(in.Tags); i++ {
		res, err := db.Exec("INSERT INTO tags(content, ctime) VALUES (?, NOW())", in.Tags[i])
		if err != nil {
			log.Printf("add tag failed tag:%s err:%v\n", in.Tags[i], err)
			continue
		}
		id, err := res.LastInsertId()
		if err != nil {
			log.Printf("get tag insert id failed:%v", err)
			continue
		}
		ids = append(ids, int32(id))
	}
	return &modify.AddTagReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Ids: ids}, nil
}

func main() {
	lis, err := net.Listen("tcp", util.ModifyServerPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	db, err = util.InitDB(false)
	if err != nil {
		log.Fatalf("failed to init db connection:%v", err)
	}
	db.SetMaxIdleConns(util.MaxIdleConns)

	kv := util.InitRedis()
	go util.ReportHandler(kv, util.ModifyServerName, util.ModifyServerPort)

	s := grpc.NewServer()
	modify.RegisterModifyServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
