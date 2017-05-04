package main

import (
	"fmt"
	"log"
	"net"
	"net/url"

	"Server/proto/common"
	"Server/proto/push"
	"Server/util"

	context "golang.org/x/net/context"
)

const (
	packID        = "com.yunxingzh.wireless"
	packName      = "东莞无线"
	androidSecret = "key=gg4oUVKSgXjEodbaZVZnNA=="
	iosSecret     = "key=1sAWYtMUZEo04fEkwA9N1Q=="
	aliasHost     = "https://api.xmpush.xiaomi.com/v2/message/alias"
	topicHost     = "https://api.xmpush.xiaomi.com/v2/message/topic"
	pushTips      = "您收到一条新消息"
	aliasType     = 0
	topicType     = 1
	androidTerm   = 0
	iosTerm       = 1
	appLauncher   = 1
	appActivity   = 2
)

type server struct{}

func genAuthStr(term int64) string {
	if term == iosTerm {
		return iosSecret
	}
	return androidSecret
}

func genHost(pushType int64) string {
	if pushType == aliasType {
		return aliasHost
	}
	return topicHost
}

func buildMipush(info *push.PushInfo) string {
	var post string
	data := url.QueryEscape(info.Content)
	desc := info.Desc
	if desc == "" {
		desc = pushTips
	}
	if info.PushType == topicType {
		post += fmt.Sprintf("topic=%s", info.Target)
	} else {
		post += fmt.Sprintf("alias=%s", info.Target)
	}

	if info.TermType == iosTerm {
		tips := url.QueryEscape(desc)
		post += fmt.Sprintf("&description=%s&bundle_id=%s&extra.badge=1&extra.sound_url=default&extra.payload=%s",
			tips, packID, data)
		if info.Extra != "" {
			post += fmt.Sprintf("&%s", info.Extra)
		}
	} else {
		title := info.Title
		if title == "" {
			title = packName
		}
		post += fmt.Sprintf("&description=%s&payload=%s&restricted_package_name=%s&title=%s&notify_type=%d&pass_through=%d&notify_id=%d&extra.notify_foreground=%d",
			desc, data, packID, title, info.NotifyType, info.Passthrough,
			info.NotifyID, info.Foreground)
		if info.NotifyEffect == appActivity {
			post += fmt.Sprintf("&extra.intent_uri=%s&extra.notify_effect=%d",
				info.Extra, info.NotifyEffect)
		}
	}

	return post
}

func (s *server) Push(ctx context.Context, in *push.PushRequest) (*common.CommReply, error) {
	log.Printf("push request uid:%d target:%s type:%d", in.Head.Uid,
		in.Info.Target, in.Info.PushType)
	auth := genAuthStr(in.Info.TermType)
	post := buildMipush(in.Info)
	host := genHost(in.Info.PushType)
	res, err := util.HTTPRequestWithHeaders(host, post,
		map[string]string{"Authorization": auth})
	if err != nil {
		log.Printf("Push HTTPRequestWithHeaders failed:%v", err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, err
	}
	log.Printf("resp:%s", res)
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func main() {
	lis, err := net.Listen("tcp", util.PushServerPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	kv := util.InitRedis()
	go util.ReportHandler(kv, util.PushServerName, util.PushServerPort)

	s := util.NewGrpcServer()
	push.RegisterPushServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
