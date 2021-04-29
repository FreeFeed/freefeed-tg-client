package socketio

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"regexp"
	"strconv"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

func (c *Connection) mainLoop() {
	packetRe := regexp.MustCompile(`^(\d)(\d)?(\d+)?(.*)`)

	reconnectInterval := InitialReconnectInterval

connectLoop:
	for {
		nextID := 1
		replyMap := make(map[int]chan<- []byte)

		c.log.Println("Trying to connect...")
		if err := c.connect(); err != nil {
			c.log.Println("Connection error:", err)
			c.log.Println("Retrying in", reconnectInterval)

			reconnectTimer := time.NewTimer(reconnectInterval)
			select {
			case <-reconnectTimer.C:
			case <-c.closeChan:
				if !reconnectTimer.Stop() {
					<-reconnectTimer.C
				}
				break connectLoop
			}

			reconnectInterval = reconnectInterval * 3 / 2
			if reconnectInterval > MaxReconnectInterval {
				reconnectInterval = MaxReconnectInterval
			}

			continue
		}

		c.log.Println("Connected!")
		reconnectInterval = InitialReconnectInterval
		c.connChan <- struct{}{}

		rcvChan := c.rcvChan()

		pingInterval := DefaultPingInterval
		pingTimer := time.NewTimer(pingInterval)

	messageLoop:
		for {
			select {
			// Full close connection
			case <-c.closeChan:
				break messageLoop

			// Send ping
			case <-pingTimer.C:
				c.log.Println("Sending ping")
				if err := wsutil.WriteClientText(c.conn, []byte{'2'}); err != nil {
					c.log.Println("Error sending ping:", err)
					break messageLoop
				}
				pingTimer.Reset(pingInterval)

			// Receiving messages
			case msg, opened := <-rcvChan:
				c.log.Println("Message received", msg != nil, opened)
				if !opened {
					// Receiving error
					c.log.Println("Receiving error")
					break messageLoop
				}

				if parts := packetRe.FindSubmatch(msg); parts == nil {
					c.log.Println("Ivnalid message received:", string(msg))
				} else if parts[1][0] == '0' {
					// Connection props message
					v := &struct{ PingInterval int }{}
					if err := json.Unmarshal(parts[4], v); err != nil {
						c.log.Println("Cannot parse message:", err)
					} else {
						pingInterval = time.Duration(v.PingInterval) * time.Millisecond
						c.log.Println("Ping interval:", pingInterval)
						if !pingTimer.Stop() {
							<-pingTimer.C
						}
						pingTimer.Reset(pingInterval)
					}
				} else if parts[1][0] == '3' {
					// Pong message, do nothing
				} else if parts[1][0] == '4' && parts[2][0] == '0' {
					// Welcome message, do nothing
				} else if parts[1][0] == '4' && parts[2][0] == '2' {
					// Incoming message
					var msg IncomingMessage
					if err := json.Unmarshal(parts[4], &msg); err != nil {
						c.log.Println("Cannot parse incoming message: ", err)
					} else {
						c.msgChan <- msg
					}
				} else if parts[1][0] == '4' && parts[2][0] == '3' {
					// Reply message
					id, _ := strconv.Atoi(string(parts[3]))
					if ch, ok := replyMap[id]; ok {
						delete(replyMap, id)
						ch <- parts[4]
					} else {
						c.log.Println("Unknown reply ID:", string(msg))
					}
				} else {
					c.log.Println("Unknown message type:", string(msg))
				}

			// Sending messages
			case req := <-c.outbox:
				id := nextID
				nextID++
				if nextID > maxMsgID {
					nextID = 1
				}

				buf := new(bytes.Buffer)
				buf.WriteByte('4')
				buf.WriteByte('2')
				buf.WriteString(strconv.Itoa(id))
				buf.Write(req.data)
				c.log.Println("Sending message:", buf.String())
				if err := wsutil.WriteClientText(c.conn, buf.Bytes()); err != nil {
					c.log.Println("Error sending message:", err)
					break messageLoop
				}

				if ch, ok := replyMap[id]; ok {
					delete(replyMap, id)
					ch <- nil
				}
				replyMap[id] = req.reply
			}
		}

		c.log.Println("Message loop was stopped")

		// Cleaning up

		c.log.Println("Closing ping timer")
		if !pingTimer.Stop() {
			<-pingTimer.C
		}

		c.log.Println("Cleaning up reply map")
		for id, ch := range replyMap {
			delete(replyMap, id)
			ch <- nil
		}

		c.log.Println("Tying to close connection")
		if err := c.conn.Close(); err != nil {
			c.log.Println("Error closing connection:", err)
		}

		// Shoul we stop the reconnect cycle?
		select {
		case <-c.closeChan:
			break connectLoop
		default:
		}

		c.log.Println("Starting connection loop over")
	}

	close(c.connChan)
}

func (c *Connection) rcvChan() <-chan []byte {
	inbox := make(chan []byte)

	go func() {
		c.log.Println("Entering receive loop")
		defer func() {
			close(inbox)
			c.log.Println("Exiting receive loop")
		}()

		for {
			c.log.Println("Waiting for message...")
			payload, err := wsutil.ReadServerText(c.conn)
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					c.log.Println("Connection closed", err)
				} else if _, ok := err.(wsutil.ClosedError); ok {
					c.log.Println("Connection closed", err)
				} else {
					c.log.Println("Error reading payload:", err)
				}
				break
			} else {
				c.log.Println("Message received:", string(payload))
				inbox <- payload
			}
		}
	}()

	return inbox
}

func (c *Connection) connect() error {
	dialCtx, cancel := context.WithTimeout(context.Background(), ConnectionTimeout)
	defer cancel()

	conn, r, _, err := ws.Dial(dialCtx, c.url)
	if err != nil {
		return err
	}

	c.conn = conn
	if r != nil {
		// TODO process it?
	}

	return nil
}
