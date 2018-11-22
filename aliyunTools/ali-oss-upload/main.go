package main

import (
	"fmt"
	"flag"
	"io/ioutil"
	"os"
	"strings"
	// "sync"
	// "time"
	"path/filepath"
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
	var sourceDir = flag.String("source", "", "Upload source dir")
	var uploadedFile = flag.String("r", "", "Remove the contents of the file and upload again.")
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

	// 获取存储空间。
	bucket, err := client.Bucket(conf.BucketName)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(-1)
	}

	filesList, err := listFiles(*sourceDir)
	if err != nil {
		fmt.Println(err)
		return
	}

	baseDir := filepath.Dir(strings.TrimSuffix(*sourceDir, "/"))

	// 获取已经上传的文件列表
	var uploadedList []string
	if *uploadedFile != "" {
		uploadedList, err = getUploaded(*uploadedFile, baseDir)
		if err != nil {
			fmt.Println(err)
			return
		}
		filesList = dupRemoval(filesList, uploadedList)
	}

	for _, file := range filesList {
		// 上传本地文件。
		localFilePath := filepath.Join(baseDir, file)

		err = bucket.PutObjectFromFile(file, localFilePath)
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(-1)
		}

		fmt.Println(localFilePath)
	}

	// var wg sync.WaitGroup
	// for _, file := range filesList {
	// 	wg.Add(1)
	// 	fmt.Println("Start time:", time.Now())
	// 	go func(file string)  {
	// 		defer wg.Add(-1)
	// 		// 上传本地文件。
	// 		err = bucket.PutObjectFromFile(file, file)
	// 		if err != nil {
	// 			fmt.Println("Error:", err)
	// 			os.Exit(-1)
	// 		}
	// 		fmt.Println("End time:", time.Now())
	// 	}(file)
	// }
	// wg.Wait()
}

func listFiles(source string) ([]string, error) {
	var filesList []string

	info, err := os.Stat(source)
	if err != nil {
		return nil, err
	}

	var baseDir string
	if info.IsDir() {
		baseDir = filepath.Base(source)
	}

	filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		var filePath string

		if baseDir != "" {
			filePath = filepath.Join(baseDir, strings.TrimPrefix(path, source))
		}

		if info.IsDir() {
			filePath += "/"
		}

		if info.IsDir() {
			return nil
		}
		filesList = append(filesList, filePath)
		return err
	})
	
	return filesList, nil
}

func getUploaded(file, baseDir string) ([]string, error) {
	f, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	l := strings.Split(string(f), "\n")
	var fileList []string
	for _, file := range l {
		rel, err := filepath.Rel(baseDir, file)
		if err != nil {
			continue
		}
		fileList = append(fileList, rel)
	}
	return fileList, nil
}

func dupRemoval(s1, s2 []string) []string {
	m := make(map[string]struct{})
	var filesList []string
	for _, s := range s1 {
		m[s] = struct{}{}
	}
	for _, s := range s2 {
		_, ok := m[s]
		if ok {
			delete(m, s)
		}
	}
	for k := range m {
		filesList = append(filesList, k)
	}
	return filesList
}