package main

import (
	"bytes"
	"flag"
	"fmt"
	gsftp "github.com/260by/tools/sftp"
	gssh "github.com/260by/tools/ssh"
	"golang.org/x/crypto/ssh"
	"github.com/pkg/sftp"
	"strings"
	"time"
	"os"
	"os/exec"
	// "sync"
)

var (
	method = flag.String("method", "", "get or put")
	user = flag.String("u", "", "User Name")
	password = flag.String("p", "", "Password")
	key = flag.String("i", "", "SSH public key")
	host = flag.String("h", "", "Host adderss")
	port = flag.Int("P", 22, "Host port")
	src = flag.String("s", "", "Source file path")
	dst = flag.String("d", "", "destination file path")
)

var sshClient *ssh.Client
var sftpClient *sftp.Client
var err error

func main() {
	t1 := time.Now()

	flag.Parse()

	if *password != "" {
		sshClient, err = gssh.Connect(*user, *host, *port, *password)
		checkERR(err)
		sftpClient, err = gsftp.Connect(*user, *host, *port, *password)
		checkERR(err)
	}
	if *key != "" {
		sshClient, err = gssh.Connect(*user, *host, *port, *key)
		checkERR(err)
		sftpClient, err = gsftp.Connect(*user, *host, *port, *key)
		checkERR(err)
	}

	if *method == "get" {
		cmd := "ls " + *src
		buf, err := gssh.Command(sshClient, cmd)
		checkERR(err)

		files := strings.Split(buf[:len(buf)-1], "\n")
		// fmt.Printf("%v", files)

		r, _ := pathExists(*dst)
		if r == false {
			s := "mkdir -p " + *dst
			cmd := exec.Command("/bin/bash", "-c", s)
			var out bytes.Buffer
			cmd.Stdout = &out
			err := cmd.Run()
			if err != nil {
				fmt.Println(err)
			}
		}

		result, err := gsftp.Get(sftpClient, *src, *dst, files)
		checkERR(err)
		if result {
			fmt.Println("copy file from remote server finished!")
		}
	} else if *method == "put" {
		// var wg sync.WaitGroup
		// hosts := []string{"192.168.1.163", "192.168.1.164"}
		// for _, h := range hosts {
		// 	wg.Add(1)
		// 	go func(h string) {
		// 		result := sftp.Put(user, password, key, h, port, src, dst)
		// 		if result {
		// 			fmt.Println("Upload file to remote finished!")
		// 		}
		// 		defer wg.Done()
		// 	}(h)
			
		// }
		// wg.Wait()

		result, err := gsftp.Put(sftpClient, *src, *dst)
		checkERR(err)
		if result {
			fmt.Println("Upload file to remote finished!")
		}
	}

	t2 := time.Since(t1)
	fmt.Println(t2)
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

func checkERR(err error)  {
	if err != nil {
		fmt.Println(err)
		return
	}
}