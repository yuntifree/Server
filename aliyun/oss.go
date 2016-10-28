package aliyun

import (
	"log"
	"strings"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

const (
	endpoint        = "oss-cn-shenzhen.aliyuncs.com"
	accessKeyID     = "LTAIOpvgiTmAKJNi"
	accessKeySecret = "apT9ttTZcedRj5bPdOlmLgvT8vM4R4"
	yuntiBucket     = "yuntinews"
	bucketURL       = "http://yuntinews.oss-cn-shenzhen.aliyuncs.com"
)

//UploadOssFile upload content to aliyun oss
func UploadOssFile(filename, content string) bool {
	client, err := oss.New(endpoint, accessKeyID, accessKeySecret)
	if err != nil {
		log.Printf("oss init failed:%v", err)
		return false
	}

	bucket, err := client.Bucket(yuntiBucket)
	if err != nil {
		log.Printf("bucket init failed:%v", err)
		return false
	}

	err = bucket.PutObject(filename, strings.NewReader(content))
	if err != nil {
		log.Printf("PutObject failed %s: %v", filename, err)
		return false
	}

	return true
}

//GenOssURL generate oss download url
func GenOssURL(filename string) string {
	return bucketURL + "/" + filename
}
