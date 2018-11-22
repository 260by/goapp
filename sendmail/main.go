package main

import (	
	"github.com/260by/tools/mail"
	"fmt"
)

func main()  {
	to := []string{"test@example.com", "hr@example.com"}
	subject := "Test Mail"
	body := "1111111"
	attach := ""
	result, err := mail.SendMail(subject, body, attach, to)
	if err != nil {
		fmt.Println(err)
	}
	if result {
		fmt.Println("Mail send is Success.")
	} else {
		fmt.Println("Mail send is Failed.")
	}
}
