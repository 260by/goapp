package main

import (
	"flag"
	"fmt"
	"os"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/260by/tools/gconfig"
)

type config struct {
	EndPoint string	`yaml:"EndPoint"`
	AccessKeyID string `yaml:"AccessKeyID"`
	AccessKeySecret string `yaml:"AccessKeySecret"`
	BucketName string `yaml:"BucketName"`
}

func main() {
	var configFile = flag.String("c", "config.yml", "Configration file path")
	flag.Parse()

	var conf config
	err := gconfig.Parse(*configFile, &conf)
	if err != nil {
		fmt.Println(err)
		return
	}

	// 创建OSSClient实例。
	client, err := oss.New(conf.EndPoint, conf.AccessKeyID, conf.AccessKeySecret)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(-1)
	}

	bucketName := conf.BucketName

	// 获取存储空间。
	bucket, err := client.Bucket(bucketName)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(-1)
	}

	// 分页列举包含指定前缀的文件。每页列举500个
	prefix := oss.Prefix("英文歌曲994首/")
	marker := oss.Marker("")
	for {
		lsRes, err := bucket.ListObjects(oss.MaxKeys(500), marker, prefix)
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(-1)
		}

		prefix = oss.Prefix(lsRes.Prefix)
		marker = oss.Marker(lsRes.NextMarker)

		fmt.Println(len(lsRes.Objects))

		if !lsRes.IsTruncated {
			break
		}
	}
}
