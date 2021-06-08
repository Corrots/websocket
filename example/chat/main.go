package main

import (
	"github.com/corrots/socket"
	"github.com/gin-gonic/gin"
	"log"
)

func main() {
	r := gin.Default()
	m := socket.New()
	defer m.Close()

	r.GET("/", func(c *gin.Context) {
		r.LoadHTMLFiles("index.html")
		//http.ServeFile(c.Writer, c.Request, "index.html")
	})

	r.GET("/ws", func(c *gin.Context) {
		m.HandleRequest(c.Writer, c.Request)
	})

	m.HandleMessage(func(s *socket.Session, msg []byte) {
		if err := m.Broadcast(msg); err != nil {
			log.Fatalf("broadcast message err: %v\n", err)
		}
	})

	r.Run(":8080")
}
