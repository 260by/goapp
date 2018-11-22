package check

import (
	"fmt"
	"time"
	"path"
	"github.com/260by/goapp/checkFile/config"
	"strconv"
	"os"
	"strings"
)

// CheckFile 检查文件是否存在
func CheckFile(t, f string, l config.Log) string {
	var file string
	var messages string
	switch t {
	case "day":
		file = f + "-" + time.Now().AddDate(0, 0, -1).Format("20060102")
		
		filePath := path.Join(replDir(l.Path.Dir), file)
		fullFilePath := addExt(filePath, l.Path.Ext)
		result, _ := pathExists(fullFilePath)
		messages = msg(result, fullFilePath)
	case "houre":
		for i := 00; i < 24; i++ {
			if i < 10 {
				file = f + "-" + time.Now().AddDate(0, 0, -1).Format("20060102") + "0" + strconv.Itoa(i)
			} else {
				file = f + "-" + time.Now().AddDate(0, 0, -1).Format("20060102") + strconv.Itoa(i)
			}
			filePath := path.Join(replDir(l.Path.Dir), file)
			fullFilePath := addExt(filePath, l.Path.Ext)
			result, _ := pathExists(fullFilePath)
			messages = messages + msg(result, fullFilePath)
		}
	}
	return messages
}

// pathExists 判断文件是否存在
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

// replDir 替换路径中的year, month, day为实际值
func replDir(dir string) string {
	year := time.Now().Format("2006")
	month := time.Now().Format("01")
	day := time.Now().AddDate(0, 0, -1).Format("02")
	p := strings.Replace(strings.Replace(strings.Replace(dir, "year", year, -1), "month", month, -1), "day", day, -1)

	return p
}

// addExt 增加文件扩展名
func addExt(filePath, ext string) (fullFilePath string) {
	// var fullFilePath string
	if ext != "" {
		fullFilePath = fmt.Sprintf("%s%s", filePath, ext)
	} else {
		fullFilePath = filePath
	}
	return fullFilePath
}

func msg(b bool, filePath string) (messages string) {
	// var messages string
	if b != true {
		messages = fmt.Sprintf("%v file is not exist\n", filePath)
	}
	return messages
}