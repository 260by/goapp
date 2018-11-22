package main

import (
	"os"
	"bufio"
	"time"
	"io"
	"fmt"
)

type fileReader struct {
	file string
	offset int64
}

// func (f *fileReader) Read(p []byte) (n int, err error) {
// 	reader, err := os.Open(f.file)
// 	defer reader.Close()
// 	if err != nil {
// 		return 0, err
// 	}
// 	reader.Seek(f.offset, 0)

// 	n, err = reader.Read(p)

// 	if err == io.EOF {
// 		time.Sleep(1 * time.Second)
// 	}
// 	f.offset += int64(n)

// 	return n, err
// }

func main()  {
	// file := &fileReader{os.Args[1], 0}
	file, _ := os.Open(os.Args[1])
	file.Seek(0, 0)
	defer file.Close()
	br := bufio.NewReader(file)
	var offset int64
	for {
		log, _, err := br.ReadLine()
		offset += int64(len(log))
		if err == io.EOF {
			continue
		}

		if err != nil {
			fmt.Println("ERR:", err)
			return
		}

		fmt.Println(string(log))
		time.Sleep(300 * time.Millisecond)
		fmt.Println("offset:", offset)
	}
}