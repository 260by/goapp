// 更新阿里云CDN指定域名证书
package main

import (
	"fmt"
	"flag"
	"io/ioutil"
	"os"
	"strings"
	"time"
	// "github.com/260by/tools/gconfig"
	"github.com/260by/tools/mail"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/cdn"
	log "github.com/sirupsen/logrus"
)

type config struct {
	Domains         []string `yaml:"Domain"`
	RegionID        string   `yaml:"RegionID"`
	AccessKeyID     string   `yaml:"AccessKeyID"`
	AccessKeySecret string   `yaml:"AccessKeySecret"`
	SSLCertDir      string   `yaml:"SSLCertDir"`
}

func init() {
	// log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	// log.SetLevel(log.InfoLevel)
}

func usage() {
	fmt.Printf(`Description: Update Aliyun cdn appoint domain ssl certificate.
Options:
`)
	flag.PrintDefaults()
}

func main() {
	// configFile := flag.String("config", "config.yaml", "Configration file path")
	day := flag.Int("day", 5, "Check whether the certificate expires in a few days.")
	flag.Usage = usage
	flag.Parse()

	conf := &config{
		Domains:         []string{"test.aliyuncs.com"},
		RegionID:        "cn-hangzhou",
		AccessKeyID:     "accesskeyid",
		AccessKeySecret: "accesskeysecret",
		SSLCertDir:      "/usr/local/etc/nginx/ssl",
	}

	// err := gconfig.Parse(*configFile, &conf)
	// if err != nil {
	// 	log.Errorln(err)
	// 	return
	// }

	client, err := cdn.NewClientWithAccessKey(conf.RegionID, conf.AccessKeyID, conf.AccessKeySecret)
	if err != nil {
		log.Errorln(err)
		return
	}
	
	var msg string

	for _, domain := range conf.Domains {
		exTime, err := checkDomainCertExpire(domain, client)
		if err != nil {
			log.Errorln(err)
			return
		}
		result, already, err := comparisonDate(exTime, *day)
		if err != nil {
			log.Errorln(err)
			return
		}

		// 指定多少天后是否过期或已经过期
		if result || already {
			if already {
				log.Infof("%s already certificate is due.", domain)
				msg += fmt.Sprintf("%s already certificate is due.\n", domain)
			} else {
				log.Infof("%s %v days later the certificate is due.", domain, *day)
				msg += fmt.Sprintf("%s %v days later the certificate is due.\n", domain, *day)
			}
			
			requestID, err := uploadCert(domain, conf.SSLCertDir, client)
			if err != nil {
				log.Errorln(err)
				msg += fmt.Sprintf("%v\n", err)
			} else {
				log.Infof("RequestID: %s Upload %s certificate is success.", requestID, domain)
				msg += fmt.Sprintf("Upload %s certificate is success.", domain)
			}
		}
	}

	// 发送邮件
	if msg != "" {
		mail.SendMail("阿里云SSL证书更新", msg, []string{}, []string{"admin@example.com", "manager@example.com"}, mail.SMTP{
			Server: "smtp.example.com",
			Port: 25,
			User: "sendmail@example.com",
			Password: "mypassword",
		})
	}
}

// 获取指定域名证书到期日期
func checkDomainCertExpire(domain string, client *cdn.Client) (string, error) {
	request := cdn.CreateDescribeDomainCertificateInfoRequest()
	request.DomainName = domain
	response, err := client.DescribeDomainCertificateInfo(request)
	if err != nil {
		return "", err
	}
	return response.CertInfos.CertInfo[0].CertExpireTime, nil
}

// 上传指定域名证书
func uploadCert(domain, sslCertDir string, client *cdn.Client) (requestID string, err error) {
	caFileName := fmt.Sprintf("%s/%s/fullchain.cer", sslCertDir, domain)
	priFileName := fmt.Sprintf("%s/%s/%s.key", sslCertDir, domain, domain)
	caFile, err := ioutil.ReadFile(caFileName)
	if err != nil {
		return
	}
	priFile, err := ioutil.ReadFile(priFileName)
	if err != nil {
		return
	}

	request := cdn.CreateSetDomainServerCertificateRequest()
	request.DomainName = domain
	request.ServerCertificateStatus = "on"
	request.ServerCertificate = string(caFile)
	request.PrivateKey = string(priFile)
	request.CertType = "upload"
	response, err := client.SetDomainServerCertificate(request)
	if err != nil {
		return
	}
	return response.RequestId, nil
}

// 检查多少天后是否到期
func comparisonDate(expireTime string, day int) (result, already bool, err error) {
	t := strings.Split(expireTime, "T")[0]

	tTime, err := time.ParseInLocation("2006-01-02", t, time.Local)
	if err != nil {
		return
	}
	nTime, err := time.ParseInLocation("2006-01-02", time.Now().AddDate(0, 0, day).Format("2006-01-02"), time.Local)
	if err != nil {
		return
	}
	if tTime.Equal(nTime) {
		result = true
	} else if tTime.Before(nTime) {  // 查检是否已经过期
		already = true
	} else {
		result = false
	}

	return
}
