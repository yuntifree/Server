package util

import (
	"strconv"
	"time"

	"gopkg.in/redis.v5"
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
