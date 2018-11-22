package main

import (
	"fmt"
	"flag"
	"github.com/260by/goapp/checkFile/config"
	"github.com/260by/goapp/checkFile/check"
	"log"
)

var (
	f string
	t string
)

func init()  {
	flag.StringVar(&f, "f", "", "Config file")
	flag.StringVar(&t, "t", "", "day or houre")
}

func main() {
	flag.Parse()
	if f == "" {
		log.Fatalln("require config file")
	}

	if err := config.LoadFile(f); err != nil {
		// fmt.Println(err)
		log.Fatalln(err)
	}

	for _, l := range config.Config.Logs {
		for _, f := range l.Path.Files {
			messages := check.CheckFile(t, f, l)
			fmt.Printf(messages)
		}
	}
}