package util

import (
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
func Report(client *redis.Client, name, addr string) {
	ts := time.Now().Unix()
	client.ZAdd(name, redis.Z{Member: addr, Score: float64(ts)})
}
