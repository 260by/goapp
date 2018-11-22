package main

import (
	// "fmt"
	"github.com/kataras/iris"
	"github.com/iris-contrib/middleware/cors"
	"github.com/kataras/iris/middleware/logger"
)

func main()  {
	app := iris.New()
	app.Logger().SetLevel("debug")
	app.Use(cors.Default())
	app.Use(logger.New())

	app.StaticWeb("/", "./static")

	app.Get("/api/student", func(ctx iris.Context)  {
		ctx.JSON(iris.Map{"Name": "keith", "Age": "34", "Email": "56119476@qq.com"})
	})

	app.Run(iris.Addr(":9000"))
}