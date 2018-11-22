package main

import (
	"flag"
	"github.com/260by/tools/zip"
	"log"
)

var (
	source string
	target string
)

func init() {
	flag.StringVar(&source, "s", "", "Archive source directory")
	flag.StringVar(&target, "d", "", "Archive filename")
}

func main() {
	flag.Parse()
	err := zip.Compress(source, target)
	if err != nil {
		log.Fatalln(err)
	}
}
