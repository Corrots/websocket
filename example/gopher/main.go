package main

import (
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/corrots/socket"
	"github.com/gin-gonic/gin"
)

type GopherInfo struct {
	ID, X, Y string
}

func main() {
	router := gin.Default()
	mrouter := socket.New()
	gophers := make(map[*socket.Session]*GopherInfo)
	lock := new(sync.Mutex)
	counter := 0

	router.GET("/", func(c *gin.Context) {
		http.ServeFile(c.Writer, c.Request, "index.html")
	})

	router.GET("/ws", func(c *gin.Context) {
		mrouter.HandleRequest(c.Writer, c.Request)
	})

	mrouter.HandleConnect(func(s *socket.Session) {
		lock.Lock()
		for _, info := range gophers {
			s.SendWithText([]byte("set " + info.ID + " " + info.X + " " + info.Y))
		}
		gophers[s] = &GopherInfo{strconv.Itoa(counter), "0", "0"}
		s.SendWithText([]byte("iam " + gophers[s].ID))
		counter++
		lock.Unlock()
	})

	mrouter.HandleDisconnect(func(s *socket.Session) {
		lock.Lock()
		mrouter.BroadcastOthers([]byte("dis "+gophers[s].ID), s)
		delete(gophers, s)
		lock.Unlock()
	})

	mrouter.HandleMessage(func(s *socket.Session, msg []byte) {
		p := strings.Split(string(msg), " ")
		lock.Lock()
		info := gophers[s]
		if len(p) == 2 {
			info.X = p[0]
			info.Y = p[1]
			mrouter.BroadcastOthers([]byte("set "+info.ID+" "+info.X+" "+info.Y), s)
		}
		lock.Unlock()
	})

	router.Run(":8080")
}
