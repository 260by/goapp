package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"io"
)

// 一次性读取
func ReadAll(filename string) string {
	f, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println(err)
	}
	return string(f)
}

// 逐行读取
func ReadLine(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		fmt.Println(err)
	}
	defer f.Close()

	buf := bufio.NewReader(f)
	i := 1
	for {
		line, err := buf.ReadBytes('\n')
		fmt.Printf("%d: %s",i, line)
		if err != nil {
			if err == io.EOF {
				break
			}
		}
		i++
	}
}

func main()  {
	file := "/home/keith/go/src/tools/test.txt"
	// s := ReadAll(file)
	// fmt.Printf("%T\n", s)
	// fmt.Println(s)
	ReadLine(file)
}