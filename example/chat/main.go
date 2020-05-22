package main

import (
	"fmt"
	"net/http"

	"github.com/corrots/socket"
	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()
	m := socket.New()

	router.GET("/", func(c *gin.Context) {
		http.ServeFile(c.Writer, c.Request, "index.html")
	})

	router.GET("/ws", func(c *gin.Context) {
		m.HandleRequest(c.Writer, c.Request)
	})

	m.HandleMessage(func(s *socket.Session, msg []byte) {
		fmt.Println(m.Broadcast(msg))
	})

	fmt.Println("err: ", router.Run(":8080"))
}
