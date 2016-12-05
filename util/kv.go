package util

import (
	"errors"
	"log"
	"strconv"
	"time"

	simplejson "github.com/bitly/go-simplejson"
	ssdb "github.com/ssdb/gossdb/ssdb"
	"gopkg.in/redis.v5"
)

const (
	userTokenSet   = "user:req:token"
	expireTime     = 3600 * 24 * 30
	expireInterval = 300
	redisHost      = "r-wz9191666aa18664.redis.rds.aliyuncs.com:6379"
	redisPasswd    = "YXZHwifiredis01server"
)

//InitRedis return initialed redis client
func InitRedis() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     redisHost,
		Password: redisPasswd,
		DB:       0,
	})
}

//Report add address to server list
func Report(client *redis.Client, name, port string) {
	ip := GetInnerIP()
	addr := ip + port
	if ip == DebugHost {
		name += ":debug"
	}
	ts := time.Now().Unix()
	client.ZAdd(name, redis.Z{Member: addr, Score: float64(ts)})
	client.ZRemRangeByScore(name, "0", strconv.Itoa(int(ts-20)))
}

//ReportHandler handle report address
func ReportHandler(kv *redis.Client, name, port string) {
	for {
		time.Sleep(time.Second * 2)
		Report(kv, name, port)
	}
}

//SetCachedToken set token in redis
func SetCachedToken(kv *redis.Client, uid int64, token string) {
	js, err := simplejson.NewJson([]byte(`{}`))
	if err != nil {
		log.Printf("RefreshCachedToken NewJson failed:%v", err)
		return
	}

	js.Set("expire", time.Now().Unix()+expireTime)
	js.Set("ts", time.Now().Unix())
	js.Set("token", token)
	data, err := js.Encode()
	if err != nil {
		log.Printf("RefreshCachedToken Encode failed:%v", err)
		return
	}
	_, err = kv.HSet(userTokenSet, strconv.Itoa(int(uid)), string(data)).Result()
	if err != nil {
		log.Printf("RefreshCachedToken HSet failed uid:%d token:%s err:%v", uid, token, err)
		return
	}
	return
}

//GetCachedToken get user's token from redis
func GetCachedToken(kv *redis.Client, uid int64) (token string, err error) {
	res, err := kv.HGet(userTokenSet, strconv.Itoa(int(uid))).Result()
	if err != nil {
		log.Printf("GetCachedToken HGet failed:%d %v", uid, err)
		return
	}
	js, err := simplejson.NewJson([]byte(res))
	if err != nil {
		log.Printf("GetCachedToken NewJson failed:%v", err)
		return
	}

	ts, _ := js.Get("ts").Int64()
	if time.Now().Unix() > ts+expireInterval {
		log.Printf("GetCachedToken cache expired:%d\n", ts)
		err = errors.New("cache expired")
		return
	}
	expire, _ := js.Get("expire").Int64()
	if time.Now().Unix() > expire {
		log.Printf("GetCachedToken token expired:%d\n", expire)
		err = errors.New("token expired")
		return
	}
	token, _ = js.Get("token").String()
	return
}

//GetSSDBVal get key-val from ssdb
func GetSSDBVal(key string) (val string, err error) {
	cli, err := ssdb.Connect("127.0.0.1", 8888)
	if err != nil {
		log.Printf("GetSSDBVal Connect failed:%v", err)
		return
	}
	defer cli.Close()
	res, err := cli.Get(key)
	if err != nil {
		log.Printf("GetSSDBVal failed:%v", err)
		return
	}
	switch res.(type) {
	default:
		return "", errors.New("key not found")
	case string:
		return res.(string), nil
	}
}

//SetSSDBVal set key-val to ssdb
func SetSSDBVal(key, val string) (err error) {
	cli, err := ssdb.Connect("127.0.0.1", 8888)
	if err != nil {
		log.Printf("GetSSDBVal Connect failed:%v", err)
		return
	}
	defer cli.Close()
	_, err = cli.Set(key, val)
	return
}
