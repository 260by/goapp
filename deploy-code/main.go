package main

import (
	"flag"
	"fmt"
	"github.com/260by/tools/gconfig"
	"github.com/260by/tools/sftp"
	"github.com/260by/tools/ssh"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type config struct {
	Project []struct {
		Name   string	`yaml:"Name"`
		Git    string	`yaml:"Git"`
		Server []string	`yaml:"Server"`
	}
	User       string	`yaml:"User"`
	PrivateKey string	`yaml:"PrivateKey"`
	UploadDir  string	`yaml:"UploadDir"`
	UnzipDir   string	`yaml:"UnzipDir"`
	DeployDir  string	`yaml:"DeployDir"`
}

func main() {
	var configFile = flag.String("c", "config.yml", "Configration file name")
	var tag = flag.String("tag", "", "Tag name")
	var project = flag.String("project", "", "Project name")
	var opt = flag.String("opt", "", "upload|release")
	flag.Parse()

	var conf config
	err := gconfig.Parse(*configFile, &conf)
	if err != nil {
		log.Fatalln(err)
	}

	// fmt.Println(conf.UploadDir)
	// os.Exit(0)
	var projectName, gitAddr string
	var serverAddr []string

	// 检测传入参数与配置文件是否相符
	for _, proj := range conf.Project {
		if proj.Name == *project {
			projectName = proj.Name
			gitAddr = proj.Git
			serverAddr = proj.Server
		}
	}

	if projectName == "" || gitAddr == "" || serverAddr == nil {
		fmt.Println("Please check configration file project name or git address or server addr")
		return
	}

	switch *opt {
	case "upload":
		downloadTag(gitAddr, *tag)
		uploadTag(*tag, serverAddr, conf)
	case "release":
		release(projectName, *tag, serverAddr, conf)
	default:
		fmt.Println("upload|release")
	}
}

func downloadTag(gitAddr, tag string) {
	if _, err := os.Stat("tmp"); os.IsNotExist(err) {
		os.Mkdir("tmp", 0755)
	}

	// 从gitlab上下载tag到本地tmp目录
	cmd := fmt.Sprintf("git archive --format=zip --remote=git@%s --prefix=%s/ --output=tmp/%s.zip %s", gitAddr, tag, tag, tag)
	_, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Printf("%s Download %s.zip is success.\n", time.Now().Format("2006-01-02 15:04:05"), tag)
	}
}

func uploadTag(tag string, server []string, conf config) {
	file := fmt.Sprintf("tmp/%s.zip", tag)
	if _, err := os.Stat(file); os.IsNotExist(err) {
		log.Fatalln("file does not exist")
	}

	var wg sync.WaitGroup
	for _, ip := range server {
		wg.Add(1)
		go func(ip string) {
			defer wg.Add(-1)

			// 上传tag至服务器
			sftpClient, err := sftp.Connect(conf.User, ip, 22, conf.PrivateKey)
			result, err := sftp.Put(sftpClient, file, conf.UploadDir)
			if err != nil {
				log.Fatalln(err)
			}
			if result {
				fmt.Printf("%s Upload file to %s success!\n", time.Now().Format("2006-01-02 15:04:05"), ip)
			}

			// 解压到指定的目录
			sshClient, err := ssh.Connect(conf.User, ip, 22, conf.PrivateKey)
			if err != nil {
				log.Fatalln(err)
			}
			cmd := fmt.Sprintf("sudo unzip -q %s/%s.zip -d %s", conf.UploadDir, tag, conf.UnzipDir)
			_, err = ssh.Command(sshClient, cmd)
			if err != nil {
				log.Fatalln(err)
			}

			defer sftpClient.Close()
			defer sshClient.Close()
			

		}(ip)
	}
	wg.Wait()
}

func release(name, tag string, server []string, conf config) {
	sourceDir := fmt.Sprintf("%s/%s", conf.UnzipDir, tag)  // zip文件解压后目录

	var wg sync.WaitGroup
	for _, ip := range server {
		wg.Add(1)

		go func(ip string) {
			defer wg.Add(-1)

			// 创建ssh连接
			sshClient, err := ssh.Connect(conf.User, ip, 22, conf.PrivateKey)
			if err != nil {
				log.Fatalln(err)
			}

			// 检查代码源目录是否存在
			cmd := fmt.Sprintf("ls %s", sourceDir)
			out, err := ssh.Command(sshClient, cmd)
			if err != nil {
				fmt.Println("\033[1;31mPlease check source code path.\033[0m")
				return
			}

			// 获取上一个版本tag
			cmd = fmt.Sprintf("ls -l %s/%s", conf.DeployDir, name)
			out, err = ssh.Command(sshClient, cmd)
			if err != nil {
				fmt.Println("\033[1;31mLast version link is not exist.\033[0m")
			} else {
				v := strings.Split(out, "\n")[0]
				lastVersion := strings.Fields(v)[10]
				fmt.Printf("\033[1;32mLast version: %s \033[0m\n", lastVersion)
			}

			// 创建软链接到源码目录
			cmd = fmt.Sprintf("sudo ln -nsf %s %s/%s", sourceDir, conf.DeployDir, name)
			_, err = ssh.Command(sshClient, cmd)
			if err != nil {
				log.Fatalln(err)
			} else {
				fmt.Printf("%s deploy %s to %s success.\n", time.Now().Format("2006-01-02 15:04:05"), tag, ip)
			}

			sshClient.Close()
		}(ip)
	}
	wg.Wait()

}
