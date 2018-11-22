package main

import "net/http"

func main()  {
	http.Handle("/", http.FileServer(http.Dir("/home/keith")))
	http.ListenAndServe(":9000", nil)
}