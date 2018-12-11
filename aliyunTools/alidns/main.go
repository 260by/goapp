package main

import (
	"fmt"
	"flag"
	"net/http"
	"io/ioutil"
	"os"
	"strings"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/alidns"
	log "github.com/sirupsen/logrus"
)

// Config 配置
type Config struct {
	RegionID        string   `yaml:"RegionID"`
	SubDomainName   string   `yaml:"DomainName"`
	AccessKeyID     string   `yaml:"AccessKeyID"`
	AccessKeySecret string   `yaml:"AccessKeySecret"`
}

func init() {
	// log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	// log.SetLevel(log.InfoLevel)
}

func usage() {
	fmt.Printf(`Description: Update bj.evenote.cn domain name record as local public network ip.
`)
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()

	config := &Config{
		RegionID:        "cn-hangzhou",
		SubDomainName:   "test.example.com",
		AccessKeyID:     "accesskeyid",
		AccessKeySecret: "accesskeysecret",
	}


	file := fmt.Sprintf("/tmp/%s", config.SubDomainName)
	result, err := pathExists(file)
	if err != nil {
		log.Fatalln(err)
	}

	publicIP, err := GetPublicNetworkIP()
	if err != nil {
		log.Fatalln(err)
	}

	if result {
		f, err := ioutil.ReadFile(file)			// 读取本地保存的解析记录
		if err != nil {
			log.Fatalln(err)
		}
		id := strings.Split(string(f), " ")[0]  // 获取本地解析记录ID
		v := strings.Split(string(f), " ")[1]	// 获取本地解析记录值

		// 判断本地保存的解析记录值是否等于公网IP
		if v == publicIP {
			log.Println("Record no change")
		} else {
			recID, err := config.UpdateDomainRecord(id, publicIP)
			if err != nil {
				log.Fatalln(err)
			} else {
				log.Printf("Update record success, RecordID: %s", recID)
			}
			
			updateLocalRecordFile(file, recID, publicIP)
		}
	} else {  // 本地保存的解析记录文件不存在
		recID, recValue, err := config.GetSubDomainNameRecordValue()
		if err != nil {
			log.Fatalln(err)
		}
		if recValue == publicIP {  // 云解析记录值等于公网IP,创建本地解析记录文件
			updateLocalRecordFile(file, recID, publicIP)
		} else { // 云解析记录值不等于公网IP时，更新云解析记录并创建本地解析记录文件
			_, err := config.UpdateDomainRecord(recID, publicIP)
			if err != nil {
				log.Fatalln(err)
			} else {
				log.Printf("Update record success, RecordID: %s", recID)
			}
			updateLocalRecordFile(file, recID, publicIP)
		}
		
	}
}

// UpdateDomainRecord 更新A记录
func (c *Config) UpdateDomainRecord(recordID, recordValue string) (id string, err error) {
	client, err := alidns.NewClientWithAccessKey(c.RegionID, c.AccessKeyID, c.AccessKeySecret)
	if err != nil {
		return
	}

	req := alidns.CreateUpdateDomainRecordRequest()
	req.RecordId = recordID
	s := strings.Split(c.SubDomainName, ".")
	req.RR = strings.Join(s[:len(s)-2], ".")
	req.Type = "A"
	req.Value = recordValue
	
	res, err := client.UpdateDomainRecord(req)
	if err != nil {
		return
	}

	return res.RecordId, nil
}

// GetSubDomainNameRecordValue 获取子域名记录值
func (c *Config) GetSubDomainNameRecordValue() (recordID, recordValue string, err error) {
	client, err := alidns.NewClientWithAccessKey(c.RegionID, c.AccessKeyID, c.AccessKeySecret)
	if err != nil {
		return
	}

	req := alidns.CreateDescribeSubDomainRecordsRequest()
	req.SubDomain = c.SubDomainName
	res, err := client.DescribeSubDomainRecords(req)
	if err != nil {
		return
	}

	record := res.DomainRecords.Record[0]

	return record.RecordId, record.Value, nil
}

// GetPublicNetworkIP 获取公网IP
func GetPublicNetworkIP() (ip string, err error) {
	response, err := http.Get("http://ip.cip.cc")
	if err != nil {
		return
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return
	}
	ip = strings.TrimSuffix(string(body), "\n")
	return
}

func updateLocalRecordFile(file, id, ip string) {
	str := fmt.Sprintf("%s %s", id, ip)
	err := ioutil.WriteFile(file, []byte(str), 0644)
	if err != nil {
		log.Fatalln(err)
	}
}

func pathExists(path string) (bool, error) {
    _, err := os.Stat(path)
    if err == nil {
        return true, nil
    }
    if os.IsNotExist(err) {
        return false, nil
    }
    return false, err
}