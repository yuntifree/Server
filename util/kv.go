package util

import (
	"log"
	"strconv"
	"time"

	simplejson "github.com/bitly/go-simplejson"
	"gopkg.in/redis.v5"
)

const (
	userTokenSet   = "user:req:token"
	expireTime     = 3600 * 24 * 30
	expireInterval = 300
)

//InitRedis return initialed redis client
func InitRedis() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
}

//Report add address to server list
func Report(client *redis.Client, name, port string) {
	ip := GetInnerIP()
	addr := ip + port
	ts := time.Now().Unix()
	client.ZAdd(name, redis.Z{Member: addr, Score: float64(ts)})
	client.ZRemRangeByScore(name, "0", strconv.Itoa(int(ts-20)))
}

//ReportHandler handle report address
func ReportHandler(name, port string) {
	kv := InitRedis()
	for {
		time.Sleep(time.Second * 2)
		Report(kv, name, port)
	}
}

//RefreshCachedToken refresh token in redis
func RefreshCachedToken(uid int64, token string) {
	js, err := simplejson.NewJson([]byte(`{}`))
	if err != nil {
		log.Printf("RefreshCachedToken NewJson failed:%v", err)
		return
	}

	js.Set("expire", time.Now().Unix()+expireTime)
	js.Set("ts", time.Now().Unix())
	data, err := js.Encode()
	if err != nil {
		log.Printf("RefreshCachedToken Encode failed:%v", err)
		return
	}
	kv := InitRedis()
	kv.HSet(userTokenSet, strconv.Itoa(int(uid)), string(data))
	return
}
