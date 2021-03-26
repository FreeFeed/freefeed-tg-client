package socketio

import (
	"encoding/json"
	"errors"
	"net"
	"time"

	"github.com/davidmz/debug-log"
)

const (
	// ConnectionTimeout is the default connection timeout
	ConnectionTimeout = 5 * time.Second
	// ReplyTimeout is the default timeout of reply messages
	ReplyTimeout = 5 * time.Second
	// DefaultPingInterval is the default keepalive ping interval
	DefaultPingInterval = 20 * time.Second

	// InitialReconnectInterval is the initial (minimal) reconnect interval
	InitialReconnectInterval = time.Second
	// MaxReconnectInterval is the maximum reconnect interval
	MaxReconnectInterval = time.Minute

	maxMsgID = 10000
)

type sendRequest struct {
	data  []byte
	reply chan []byte
}

// Connection is the single connection to SocketIO server.
type Connection struct {
	url  string
	conn net.Conn
	log  debug.Logger

	connChan chan struct{}
	msgChan  chan IncomingMessage

	closeChan chan struct{}
	outbox    chan sendRequest
}

// Open creates a new connection to SocketIO server
func Open(url string, options ...Option) *Connection {
	c := &Connection{
		url:      url,
		connChan: make(chan struct{}, 1),
		msgChan:  make(chan IncomingMessage),

		closeChan: make(chan struct{}),
		outbox:    make(chan sendRequest, 1),
	}
	Options(append(
		Options{WithLogger(debug.NewLogger("socketio"))},
		options...,
	)).apply(c)

	go c.mainLoop()

	return c
}

// Close closes the connection and stops the reconnect cycle.
func (c *Connection) Close() {
	close(c.closeChan)
}

// Connected returns the channel of 'connected' events.
func (c *Connection) Connected() <-chan struct{} { return c.connChan }

// Messages returns the channel of incoming messages.
func (c *Connection) Messages() <-chan IncomingMessage { return c.msgChan }

// Close closes the connection. The closed connection cannot be reopened.
// func (c *Connection) Close() {
// 	close(c.closeChan)
// 	close(c.connChan)
// }

// Send sending message to the server and waiting for the response.
func (c *Connection) Send(command string, payload interface{}) ([]byte, error) {
	data, err := json.Marshal([]interface{}{command, payload})
	if err != nil {
		return nil, err
	}
	c.log.Println("Sending data:", string(data))
	reply := make(chan []byte)
	c.outbox <- sendRequest{data, reply}

	timer := time.NewTimer(ReplyTimeout)
	var replyData []byte

	select {
	case replyData = <-reply:
	case <-timer.C:
		go func() { <-reply }()
		return nil, errors.New("Reply timeout")
	}

	if !timer.Stop() {
		<-timer.C
	}

	if replyData == nil {
		return nil, errors.New("Session was prematurely closed")
	}
	return replyData, nil
}

// IncomingMessage represents an incoming message from SocketIO server
type IncomingMessage struct {
	Type    string
	Payload json.RawMessage
}

// UnmarshalJSON implements json.Unmarshaler for IncomingMessage.
func (m *IncomingMessage) UnmarshalJSON(data []byte) error {
	var v []json.RawMessage
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	if len(v) != 2 {
		return errors.New("invalid message length")
	}
	if err := json.Unmarshal(v[0], &m.Type); err != nil {
		return err
	}
	m.Payload = v[1]

	return nil
}
