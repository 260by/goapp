// 查询阿里云监控报警历史写入csv文件，并发送邮件
package main

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/services/cms"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/260by/tools/gconfig"
	"github.com/260by/tools/mail"
	"fmt"
	"flag"
	"time"
	"encoding/json"
	"encoding/csv"
	"io/ioutil"
	"os"
	"strings"
	"strconv"
)

const (
	timeFormatSecond string = "2006-01-02 15:04:05"
)

var (
	configFile string
	startTime string
	endTime string
	csvList []string
)

type config struct {
	RegionID string
	AccessKey string
	SecretKey string
	PageSize int
	FileDir string
	SMTP struct {
		Server string
		Port int
		User string
		Password string
	}
	SendTO []string
	MailTitle string
	ECSListFile string
	RDSListFile string
	KVStoreListFile string
}

var conf = config{}

func init()  {
	flag.StringVar(&configFile, "config", "config.yaml", "Configration file name")
	flag.StringVar(&startTime, "start", "", "Start Time e.g: 2018-01-01 00:00:00")
	flag.StringVar(&endTime, "end", "", "End Time e.g: 2018-01-07 23:59:59")
}

func main() {
	flag.Parse()

	if err := gconfig.Parse(configFile, &conf); err != nil {
		panic(err)
	}

	client, err := cms.NewClientWithAccessKey(conf.RegionID, conf.AccessKey, conf.SecretKey)
	if err != nil {
		panic(err)
	}

	request := cms.CreateListAlarmHistoryRequest()
	request.Size = requests.NewInteger(conf.PageSize)
	request.StartTime = startTime
	request.EndTime = endTime
	// request.Cursor = "100"

	response, err := client.ListAlarmHistory(request)
	if err != nil {
		panic(err)
	}

	fileName := fmt.Sprintf("%s/AlarmHistory_%v.csv", conf.FileDir, time.Now().Unix())
	f, err := os.Create(fileName)
	if err != nil {
		panic(err)
	}
	w := csv.NewWriter(f)

	title := []string{
		"Alarm Time",
		"Monitor Item",
		"Alarm Duration(sec)",
		"Group",
		"Product Name",
		"Instance Name",
		"Instance ID",
		"Current Value(%)",
		"Status"}
	
	w.Write(title)

	for {
		if response.Cursor != "" {
			response, err = client.ListAlarmHistory(request)
			if err != nil {
				panic(err)
			}
			if response.Success && response.Code == "200" {
				l := displayAlarmHistoryItem(response)
				for _, v := range l {
					w.Write(v)
				}
			}
			request.Cursor = response.Cursor
		} else {
			l := displayAlarmHistoryItem(response)
			for _, v := range l {
				w.Write(v)
			}
			break
		}
	}
	w.Flush()
	csvList = append(csvList, fileName)

	subject := fmt.Sprintf("%s%s_%s", conf.MailTitle, strings.Split(startTime, " ")[0], strings.Split(endTime, " ")[0])
	body := ""
	to := conf.SendTO
	attach := csvList

	smtpServer := mail.SMTP{}
	smtpServer = conf.SMTP
	ok, err := mail.SendMail(subject, body, attach, to, smtpServer)
	if err != nil {
		panic(err)
	}
	if ok {
		fmt.Println("Send Mail is OK")
	}

}

func queryInstance(instanceID, filename string) (name, group string) {
	dat, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	r := csv.NewReader(strings.NewReader(string(dat)))

	records, e := r.ReadAll()
	if e != nil {
		panic(e)
	}
	for _, v := range records {
		if v[0] == instanceID {
			name = v[1]
			group = v[2]
			break
		}
	}

	return name, group
}

func displayAlarmHistoryItem(response *cms.ListAlarmHistoryResponse) [][]string {
	var item [][]string
	for _, alarmList := range response.AlarmHistoryList.AlarmHistory {
		if ( alarmList.MetricName == "vm.MemoryUtilization#60" || 
			alarmList.MetricName == "CPUUtilization#60" || 
			alarmList.MetricName == "CpuUsage#60" || 
			alarmList.MetricName == "MemoryUsage#60") && alarmList.State == "ALARM" {
				
				var l []string
				var mapDimension map[string]string
				if err := json.Unmarshal([]byte(alarmList.Dimension), &mapDimension); err != nil {
					panic(err)
				}
				instanceID, _ := mapDimension["instanceId"]

				var instanceName string
				var instanceGroup string
				if alarmList.Namespace == "acs_ecs" {
					instanceName, instanceGroup = queryInstance(instanceID, conf.ECSListFile)
				} else if alarmList.Namespace == "acs_rds" {
					instanceName, instanceGroup = queryInstance(instanceID, conf.RDSListFile)
				} else if alarmList.Namespace == "acs_kvstore" {
					instanceName, instanceGroup = queryInstance(instanceID, conf.KVStoreListFile)
				} else {
					instanceName = alarmList.InstanceName
				}

				if instanceName != "" {
					l = append(l, 
						time.Unix(int64(alarmList.AlarmTime)/1000, 0).Format(timeFormatSecond),
						alarmList.MetricName,
						strconv.Itoa(alarmList.LastTime/1000),
						instanceGroup,
						alarmList.Namespace,
						instanceName,
						instanceID,
						alarmList.Value,
						alarmList.State)
	
					item = append(item, l)
				}
				/*
				l = append(l, 
					time.Unix(int64(alarmList.AlarmTime)/1000, 0).Format(timeFormatSecond),
					alarmList.MetricName,
					strconv.Itoa(alarmList.LastTime/1000),
					instanceGroup,
					alarmList.Namespace,
					instanceName,
					instanceID,
					alarmList.Value,
					alarmList.State)

				item = append(item, l)
				*/
		}
	}
	return item
}