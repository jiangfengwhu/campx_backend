package main

import (
	"github.com/gin-gonic/gin"
)

func init() {
	initConf()
	dial()
}

func main() {
	if globalConf.Production {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()
	hub := newHub()
	go hub.run()
	v1 := r.Group("v1")
	v1.POST("/additem", addItem)
	v1.GET("/latest", latestItems)
	v1.PUT("/mark", markFlag)
	v1.PUT("/want", wanted)
	v1.GET("/one/:id", findOne)
	v1.GET("/anynew", anynew)
	v1.GET("/series", getSeries)
	v1.GET("/sone/:id", getSone)
	v1.GET("/anochat", func(c *gin.Context) {
		serveWs(hub, c)
	})
	r.Run(":3000") // listen and serve on 0.0.0.0:3000
}
