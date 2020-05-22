package socket

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Session struct {
	Request *http.Request
	Keys    map[string]interface{}
	conn    *websocket.Conn
	output  chan *envelope
	manager *Manager
	open    bool
	rwmutex *sync.RWMutex
}

func (s *Session) sendToSocket(msg *envelope) error {
	if s.isClosed() {
		return errors.New("tried to send to a closed session")
	}
	return s.conn.WriteMessage(msg.t, msg.msg)
}

func (s *Session) readFromSocket() {
	if s.isClosed() {
		s.manager.errorHandler(s, errors.New("tried to read from a closed session"))
		return
	}

	for {
		t, msg, err := s.conn.ReadMessage()
		if err != nil {
			s.manager.errorHandler(s, fmt.Errorf("read from a session err: %v\n", err))
			break
		}
		if t == websocket.TextMessage {
			s.manager.messageHandler(s, msg)
		}
		if t == websocket.BinaryMessage {
			s.manager.messageBinaryHandler(s, msg)
		}
	}
}

func (s *Session) sendToChan(message *envelope) {
	if s.isClosed() {
		s.manager.errorHandler(s, errors.New("session is closed"))
		return
	}
	select {
	case s.output <- message:
	default:
		s.manager.errorHandler(s, errors.New("session buffer is full"))
		return
	}
}

func (s *Session) SendWithText(msg []byte) error {
	return s.send(&envelope{t: websocket.TextMessage, msg: msg})
}

func (s *Session) SendWithBinary(msg []byte) error {
	return s.send(&envelope{t: websocket.BinaryMessage, msg: msg})
}

func (s *Session) SendCloseSignal() error {
	return s.send(&envelope{t: websocket.CloseMessage, msg: []byte{}})
}

func (s *Session) SendCloseWithMsg(msg []byte) error {
	return s.send(&envelope{t: websocket.CloseMessage, msg: msg})
}

func (s *Session) send(message *envelope) error {
	if s.isClosed() {
		return errors.New("session is closed")
	}
	s.output <- message
	return nil
}

func (s *Session) readFromChan() {
	ticker := time.NewTicker(s.manager.Config.HeartbeatRate)
	defer ticker.Stop()

loop:
	for {
		select {
		case m, ok := <-s.output:
			if !ok {
				break loop
			}
			err := s.sendToSocket(m)
			if err != nil {
				s.manager.errorHandler(s, err)
				break loop
			}
		case <-ticker.C:
			err := s.ping()
			if err != nil {
				s.manager.errorHandler(s, err)
				break loop
			}
		}
	}
}

func (s *Session) ping() error {
	return s.send(&envelope{t: websocket.TextMessage, msg: []byte{}})
}

func (s *Session) isClosed() bool {
	s.rwmutex.RLock()
	defer s.rwmutex.RUnlock()
	return !s.open
}

func (s *Session) close() {
	s.rwmutex.Lock()
	s.open = false
	s.conn.Close()
	close(s.output)
	s.rwmutex.Unlock()
}
