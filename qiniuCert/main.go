// 七牛云SSL证书管理，支持查询到期证书，上传已到期证书并更新CDN配置，删除已到期证书
// 例:
//   1. 查询所有已到期证书
//      ./qiniu -config config.yaml -env test -day 0 -mothod list
//   2. 查询2天后到期证书
//      ./qiniu -config config.yaml -env test -day 2 -mothod list
//   3. 查询2天前到期证书
//      ./qiniu -config config.yaml -env test -day -2 -mothod list
//   4. 上传2天后到期证书并更新CDN配置
//      ./qiniu -config config.yaml -env test -day 2 -mothod upload
//   5. 删除所有已到期证书
//      ./qiniu -config config.yaml -env test -mothod delete
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"io/ioutil"
	"time"
	"github.com/qiniu/api.v7/auth/qbox"
	"github.com/tidwall/gjson"
	"github.com/260by/tools/gconfig"
	log "github.com/sirupsen/logrus"
)

type Cert struct {
	CertID string
	Name string
	CommonName string
	ExpiryTime string
}

type config struct {
	Test struct {
		AccessKey string
		SecretKey string
		SSLCertDir string
	}
	Stable struct {
		AccessKey string
		SecretKey string
		SSLCertDir string
	}
}

const (
	timeFormatDay = "2006-01-02"
	hostname = "api.qiniu.com"
)

var cert = Cert{}
var mac = qbox.Mac{}
var conf = config{}
var sslCertDir string

var (
	configFilename = flag.String("config", "config.yml", "Confitration file")
	env = flag.String("env", "", "test or stable")
	day = flag.Int("day", 2, "Update the SSL certificate that expires a few days later,Value 0 update all expired SSL certificates")
	mothod = flag.String("mothod", "upload", "[upload|delete|list] expires SSL certificate")
)

func init()  {
	// log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	// log.SetLevel(log.InfoLevel)
}

func main()  {
	flag.Parse()

	err := gconfig.Parse(*configFilename, &conf)
	if err != nil {
		fmt.Println("ERROR: ", err)
		os.Exit(1)
	}

	switch *env {
	case "test":
		mac.AccessKey = conf.Test.AccessKey
		mac.SecretKey = []byte(conf.Test.SecretKey)
		sslCertDir = conf.Test.SSLCertDir
	case "stable":
		mac.AccessKey = conf.Stable.AccessKey
		mac.SecretKey = []byte(conf.Stable.SecretKey)
		sslCertDir = conf.Stable.SSLCertDir
	}

	switch *mothod {
	case "upload":
		certs, err := getCertsList(*day)
		if err != nil {
			log.Errorln(err)
			os.Exit(1)
		}
		for _, v := range certs {
			secretName := fmt.Sprintf("%s-%s", v.CommonName, time.Now().Format("20060102"))
			uploadCode, certID, err := uploadCert(secretName, v.CommonName)
			if err != nil {
				log.Errorf("Upload %s certificate: %s", v.CommonName, err)
			}
			if uploadCode == 200 {
				log.Infof("Upload %s certificate is Success", v.CommonName)
				updateCode, err := updateCert(v.CommonName, certID)
				if updateCode == 400030 || updateCode == 200 {	// 400030正在处理中
					log.Infof("Update %s certificate is Success", v.CommonName)
					continue
				}
				if err != nil {
					log.Errorf("Update %s certificate: %s", v.CommonName, err)
				}
			}
		}
	case "list":
		certs, err := getCertsList(*day)
		if err != nil {
			log.Errorln(err)
			os.Exit(1)
		}
		for _, v := range certs {
			fmt.Println("CertID: ", v.CertID)
			fmt.Println("Name: ", v.Name)
			fmt.Println("Common Name: ", v.CommonName)
			fmt.Println("Expiry Time: ", v.ExpiryTime)
			fmt.Println("----------------------------------------")
		}
	case "delete":
		delExpireCert()
	}
}

func delExpireCert() {
	certs, err := getCertsList(0)
	if err != nil {
		log.Errorln(err)
	}
	for _, v := range certs {
		deleteCode, err := deleteCert(v.CertID)
		if deleteCode == 400611 {	// 证书已绑定域名，不能删除
			log.Warnf("%s certificate has been bound to the domain name", v.CommonName)
			continue
		}
		if err != nil {
			log.Errorf("Delete %s certificate: %s", v.CommonName, err)
		}
		if deleteCode == 200 {
			log.Infof("Delete %s certificate is Success", v.CommonName)
		}
	}
}

func uploadCert(name, commonName string) (code int64,certID string, err error) {
	uri := "/sslcert"
	url := fmt.Sprintf("http://%s%s", hostname, uri)

	caFileName := fmt.Sprintf("%s/%s/fullchain.cer", sslCertDir, commonName)
	priFileName := fmt.Sprintf("%s/%s/%s.key", sslCertDir, commonName,commonName)
	caFile, err := ioutil.ReadFile(caFileName)
	if err != nil {
		return
	}
	priFile, err := ioutil.ReadFile(priFileName)
	if err != nil {
		return
	}

	str := map[string]string{"name": name,
		"common_name": commonName,
		"ca": string(caFile),
		"pri": string(priFile),
	}

	postCertStr, _ := json.Marshal(str)

	// fmt.Println(string(postCertStr))
	requestBody := bytes.NewReader(postCertStr)

	client := &http.Client{}
	reqest, err := http.NewRequest("POST", url, requestBody)
	if err != nil {
		return
	}
	token, err := mac.SignRequest(reqest)
	if err != nil {
		return
	}
	auth := fmt.Sprintf("QBox %s", token)
	reqest.Header.Add("Authorization", auth)
	reqest.Header.Add("Content-Type", "application/json")

	response, _ := client.Do(reqest)
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return
	}

	code = gjson.Get(string(body), "code").Int()
	certID = gjson.Get(string(body), "certID").String()
	e := gjson.Get(string(body), "error").String()
	if e != "" {
		err = errors.New(e)
		return
	}
	
	return code, certID, err
}

// day为0获取所有已过期证书，day大于0获取几天后过期证书,day小于0获取几天前过期证书
func getCertsList(day int) (c []Cert, err error) {
	var certs = []Cert{}

	now := time.Now()
	d, _ := time.ParseDuration("24h")
	agoOrLaterDay := now.Add(d * time.Duration(day)).Format(timeFormatDay)

	uri := "/sslcert?marker=&limit=100"
	url := fmt.Sprintf("http://%s%s", hostname, uri)

	client := &http.Client{}
	reqest, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	token, err := mac.SignRequest(reqest)
	if err != nil {
		return
	}
	auth := fmt.Sprintf("QBox %s", token)
	reqest.Header.Add("Authorization", auth)
	reqest.Header.Add("Content-Type", "application/json")

	response, err := client.Do(reqest)
	if err != nil {
		return
	}
	// defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return
	}

	certsInfo := gjson.Get(string(body), "certs")
	e := gjson.Get(string(body), "error").String()
	if e != "" {
		err = errors.New(e)
		return
	}
	for _, c := range certsInfo.Array() {
		expiryTime := time.Unix(int64(c.Get("not_after").Int()), 0).Format(timeFormatDay)
		t1, err := time.Parse(timeFormatDay, agoOrLaterDay)
		t2, err := time.Parse(timeFormatDay, expiryTime)

		if err == nil && t1.Equal(t2) {
			cert.CertID = c.Get("certid").String()
			cert.Name = c.Get("name").String()
			cert.CommonName = c.Get("common_name").String()
			cert.ExpiryTime = expiryTime
			certs = append(certs, cert)
		}

		if day == 0 && t1.After(t2) {
			cert.CertID = c.Get("certid").String()
			cert.Name = c.Get("name").String()
			cert.CommonName = c.Get("common_name").String()
			cert.ExpiryTime = expiryTime
			certs = append(certs, cert)
		}
	}
	return certs, err
}

func deleteCert(certID string) (code int64, err error) {
	uri := fmt.Sprintf("/sslcert/%s", certID)
	url := fmt.Sprintf("http://%s%s", hostname, uri)

	client := &http.Client{}
	reqest, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return
	}
	token, err := mac.SignRequest(reqest)
	if err != nil {
		return
	}
	auth := fmt.Sprintf("QBox %s", token)
	reqest.Header.Add("Authorization", auth)
	reqest.Header.Add("Content-Type", "application/json")

	response, _ := client.Do(reqest)
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return
	}

	code = gjson.Get(string(body), "code").Int()
	e := gjson.Get(string(body), "error").String()
	if e != "" {
		err = errors.New(e)
		return
	}
	return code, err
}

func updateCert(name, certID string) (code int64, err error) {
	uri := fmt.Sprintf("/domain/%s/httpsconf", name)
	url := fmt.Sprintf("http://%s%s", hostname, uri)

	str := map[string]interface{}{"certid": certID,
		"forceHttps": false,
	}

	postCertStr, _ := json.Marshal(str)
	requestBody := bytes.NewReader(postCertStr)

	client := &http.Client{}
	reqest, err := http.NewRequest("PUT", url, requestBody)
	if err != nil {
		return
	}
	token, err := mac.SignRequest(reqest)
	if err != nil {
		return
	}
	auth := fmt.Sprintf("QBox %s", token)
	reqest.Header.Add("Authorization", auth)
	reqest.Header.Add("Content-Type", "application/json")

	response, _ := client.Do(reqest)
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return
	}

	code = gjson.Get(string(body), "code").Int()
	e := gjson.Get(string(body), "error").String()
	if e != "" {
		err = errors.New(e)
		return
	}
	return code, err
}