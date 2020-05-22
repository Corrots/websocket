package socket

import (
	"errors"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type filterFunc func(*Session) bool
type handleMessageFunc func(*Session, []byte)
type handleSessionFunc func(*Session)
type HandleErrorFunc func(*Session, error)
type HandleCloseFunc func(code int, text string) error

var errInstanceClosed = errors.New("manager instance is closed")

type Manager struct {
	Config                   *Config
	Upgrader                 *websocket.Upgrader
	messageHandler           handleMessageFunc
	messageSentHandler       handleMessageFunc
	messageBinaryHandler     handleMessageFunc
	messageSentBinaryHandler handleMessageFunc
	errorHandler             HandleErrorFunc
	closeHandler             HandleCloseFunc
	connectHandler           handleSessionFunc
	disconnectHandler        handleSessionFunc
	pongHandler              handleSessionFunc
	hub                      *hub
}

func New() *Manager {
	upgrader := &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}

	hub := newHub()
	go hub.run()

	return &Manager{
		Config:                   newConfig(),
		Upgrader:                 upgrader,
		messageHandler:           func(*Session, []byte) {},
		messageSentHandler:       func(*Session, []byte) {},
		messageBinaryHandler:     func(*Session, []byte) {},
		messageSentBinaryHandler: func(*Session, []byte) {},
		errorHandler:             func(*Session, error) {},
		closeHandler:             nil,
		connectHandler:           func(*Session) {},
		disconnectHandler:        func(*Session) {},
		pongHandler:              func(*Session) {},
		hub:                      hub,
	}
}

func (m *Manager) HandleMessage(fn func(*Session, []byte)) {
	m.messageHandler = fn
}

func (m *Manager) HandleSentMessage(fn func(*Session, []byte)) {
	m.messageSentHandler = fn
}

func (m *Manager) HandleMessageBinary(fn func(*Session, []byte)) {
	m.messageBinaryHandler = fn
}

func (m *Manager) HandleSentMessageBinary(fn func(*Session, []byte)) {
	m.messageSentBinaryHandler = fn
}

func (m *Manager) HandleError(fn func(*Session, error)) {
	m.errorHandler = fn
}

func (m *Manager) HandleClose(fn func(code int, text string) error) {
	if fn != nil {
		m.closeHandler = fn
	}
}

func (m *Manager) HandleConnect(fn func(*Session)) {
	m.connectHandler = fn
}

func (m *Manager) HandleDisconnect(fn func(*Session)) {
	m.disconnectHandler = fn
}

func (m *Manager) HandlePong(fn func(*Session)) {
	m.pongHandler = fn
}

func (m *Manager) HandleRequest(w http.ResponseWriter, r *http.Request) error {
	return m.HandleRequestWithKeys(w, r, nil)
}

func (m *Manager) HandleRequestWithKeys(w http.ResponseWriter, r *http.Request, keys map[string]interface{}) error {
	conn, err := m.Upgrader.Upgrade(w, r, w.Header())
	if err != nil {
		return err
	}

	session := &Session{
		Request: r,
		Keys:    keys,
		conn:    conn,
		output:  make(chan *envelope, m.Config.MessageBufferSize),
		manager: m,
		open:    true,
		rwmutex: &sync.RWMutex{},
	}

	m.hub.register <- session
	m.connectHandler(session)

	go session.readFromChan()
	session.readFromSocket()

	if !m.hub.closed() {
		m.hub.unregister <- session
	}
	session.close()
	m.disconnectHandler(session)
	return nil
}

func (m *Manager) Broadcast(msg []byte) error {
	if m.hub.closed() {
		return errInstanceClosed
	}
	m.hub.broadcast <- &envelope{t: websocket.TextMessage, msg: msg}
	return nil
}

func (m *Manager) BroadcastFilter(msg []byte, fn func(*Session) bool) error {
	if m.hub.closed() {
		return errInstanceClosed
	}
	m.hub.broadcast <- &envelope{t: websocket.TextMessage, msg: msg, filter: fn}
	return nil
}

func (m *Manager) BroadcastOthers(msg []byte, s *Session) error {
	return m.BroadcastFilter(msg, func(q *Session) bool {
		return s != q
	})

}

func (m *Manager) BroadcastMultiple(msg []byte, sessions []Session) error {
	if m.hub.closed() {
		return errInstanceClosed
	}
	for _, sess := range sessions {
		err := sess.SendWithText(msg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) BroadcastBinary(msg []byte) error {
	if m.hub.closed() {
		return errInstanceClosed
	}
	m.hub.broadcast <- &envelope{t: websocket.BinaryMessage, msg: msg}
	return nil
}

func (m *Manager) Close() error {
	if m.hub.closed() {
		return errInstanceClosed
	}
	m.hub.exit <- &envelope{t: websocket.CloseMessage, msg: []byte{}}
	return nil
}

func (m *Manager) Len() int {
	return m.hub.len()
}
