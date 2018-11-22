// 查询阿里云数据库慢查询日志统计，写入csv文件并发送邮件
package main

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/services/rds"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/260by/tools/gconfig"
	"github.com/260by/tools/mail"
	"fmt"
	"strconv"
	"encoding/csv"
	"os"
	"time"
	"flag"
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
	SortKey string
	PageSize int
	Rds []struct {
		Name string
		DBInstanceID string
	}
	FileDir string
	SMTP struct {
		Server string
		Port int
		User string
		Password string
	}
	SendTO []string
	MailTitle string
}

var Config = config{}

func init()  {
	flag.StringVar(&configFile, "config", "config.yaml", "Configration file name")
	flag.StringVar(&startTime, "start", "", "Start Time e.g: 2018-01-01")
	flag.StringVar(&endTime, "end", "", "End Time e.g: 2018-01-07")
}

func main()  {
	t1 := time.Now()
	flag.Parse()

	// if err := loadFile(configFile); err != nil {
	// 	panic(err)
	// }
	
	if err := gconfig.Parse(configFile, &Config); err != nil {
		panic(err)
	}

	client, err := rds.NewClientWithAccessKey(
		Config.RegionID,
		Config.AccessKey,
		Config.SecretKey)
	if err != nil {
		panic(err)
	}

	for _, v := range Config.Rds {
		request := rds.CreateDescribeSlowLogsRequest()
		// request.DBInstanceId = Config.DBInstanceID
		request.DBInstanceId = v.DBInstanceID
		request.StartTime = fmt.Sprintf("%s%s", startTime, "Z")
		request.EndTime = fmt.Sprintf("%s%s", endTime, "Z")
		request.SortKey = Config.SortKey
		request.PageSize = requests.NewInteger(Config.PageSize)
		// request.PageNumber = "2"
	
		response, err := client.DescribeSlowLogs(request)
		if err != nil {
			panic(err)
		}
	
		pageSize, _ := strconv.Atoi(fmt.Sprintf("%s", request.PageSize))
		page := (response.TotalRecordCount + pageSize - 1) / pageSize
	
		if _, err := os.Stat(Config.FileDir); os.IsNotExist(err) {
			e := os.MkdirAll(Config.FileDir, 0755)
			if e != nil {
				panic(e)
			}
		}

		fileName := fmt.Sprintf("%s/%s_%s_%s_slowlog.csv", Config.FileDir, v.Name, startTime, endTime)
		f, err := os.Create(fileName)
		if err != nil {
			panic(err)
		}
		w := csv.NewWriter(f)
		title := []string{
			"CreateTime",
			"DBName",
			"MySQLTotalExecutionCounts",
			"MySQLTotalExecutionTimes",
			"MaxExecutionTime",
			"MaxLockTime",
			"ParseTotalRowCounts",
			"ParseMaxRowCount",
			"ReturnTotalRowCounts",
			"ReturnMaxRowCount",
			"SQLText"}
		w.Write(title)
	
		for i := 1; i <= page; i++ {
			request.PageNumber = requests.NewInteger(i)
			response, err := client.DescribeSlowLogs(request)
			if err != nil {
				panic(err)
			}
	
			sqlSlow := response.Items.SQLSlowLog
			
			for _, l := range sqlSlow {
				var s []string
				s = append(s, 
					l.CreateTime,
					l.DBName,
					strconv.Itoa(l.MySQLTotalExecutionCounts),
					strconv.Itoa(l.MySQLTotalExecutionTimes),
					strconv.Itoa(l.MaxExecutionTime),
					strconv.Itoa(l.MaxLockTime),
					strconv.Itoa(l.ParseTotalRowCounts),
					strconv.Itoa(l.ParseMaxRowCount),
					strconv.Itoa(l.ReturnTotalRowCounts),
					strconv.Itoa(l.ReturnMaxRowCount),
					l.SQLText)

				w.Write(s)
			}
		}
		csvList = append(csvList, fileName)
		w.Flush()
	}

	subject := Config.MailTitle
	body := ""
	to := Config.SendTO
	attach := csvList

	// sendMail(subject, body, attach, to)
	smtpServer := mail.SMTP{}
	smtpServer = Config.SMTP
	ok, err := mail.SendMail(subject, body, attach, to, smtpServer)
	if err != nil {
		panic(err)
	}
	if ok {
		fmt.Println("Send Mail is OK")
	}
	
	fmt.Println(time.Since(t1))
}
