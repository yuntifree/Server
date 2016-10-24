package util

import "gopkg.in/redis.v5"

//InitRedis return initialed redis client
func InitRedis() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
}
