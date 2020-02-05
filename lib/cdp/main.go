package cdp

import (
	"context"
	"encoding/json"
	"errors"
	"io"

	"github.com/gorilla/websocket"
	"github.com/ysmood/kit"
)

// Client is a chrome devtools protocol connection instance.
// To enable debug log, set env "debug_cdp=true".
type Client struct {
	messages map[uint64]*Message
	chReq    chan *Message
	chRes    chan *Message
	chEvent  chan *Message
	chFatal  chan error
	count    uint64
}

// Message ...
type Message struct {
	// Request
	ID        uint64      `json:"id"`
	SessionID string      `json:"sessionId,omitempty"`
	Method    string      `json:"method"`
	Params    interface{} `json:"params,omitempty"`

	// Response
	Result json.RawMessage `json:"result,omitempty"`
	Error  *Error          `json:"error,omitempty"`

	callback chan *Message
}

// Object is the json object
type Object map[string]interface{}

// Array is the json array
type Array []interface{}

// Error ...
type Error struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data"`
}

// Error ...
func (e *Error) Error() string {
	return kit.MustToJSON(e)
}

// New creates a cdp connection, the url should be something like http://localhost:9222
func New(ctx context.Context, url string) (*Client, error) {
	cdp := &Client{
		messages: map[uint64]*Message{},
		chReq:    make(chan *Message),
		chRes:    make(chan *Message),
		chEvent:  make(chan *Message),
		chFatal:  make(chan error),
	}

	wsURL, err := GetWebSocketDebuggerURL(url)
	if err != nil {
		return nil, err
	}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return nil, err
	}

	go cdp.close(ctx, conn)

	go cdp.handleReq(ctx, conn)

	go cdp.handleRes(ctx, conn)

	return cdp, nil
}

func (cdp *Client) handleReq(ctx context.Context, conn *websocket.Conn) {
	for ctx.Err() == nil {
		select {
		case msg := <-cdp.chReq:
			msg.ID = cdp.id()
			data, err := json.Marshal(msg)
			if err != nil {
				cdp.chFatal <- err
				continue
			}
			debug(data)
			err = conn.WriteMessage(websocket.TextMessage, data)
			if err != nil {
				cdp.chFatal <- err
				return
			}
			cdp.messages[msg.ID] = msg

		case msg := <-cdp.chRes:
			if msg.ID == 0 {
				cdp.chEvent <- msg
				continue
			}

			req, has := cdp.messages[msg.ID]

			if !has {
				cdp.chFatal <- errors.New("[cdp] request not found: " + kit.MustToJSON(msg))
				continue
			}

			delete(cdp.messages, msg.ID)

			req.callback <- msg
		}
	}
}

func (cdp *Client) handleRes(ctx context.Context, conn *websocket.Conn) {
	for ctx.Err() == nil {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			if err != io.EOF {
				cdp.chFatal <- err
			}
			return
		}
		debug(data)

		if msgType == websocket.TextMessage {
			var msg Message
			err = json.Unmarshal(data, &msg)
			if err != nil {
				cdp.chFatal <- err
				continue
			}

			cdp.chRes <- &msg
		}
	}
}

// Event will emit chrome devtools protocol events
func (cdp *Client) Event() chan *Message {
	return cdp.chEvent
}

// Fatal will emit fatal errors
func (cdp *Client) Fatal() chan error {
	return cdp.chFatal
}

// Call a method and get its response
func (cdp *Client) Call(ctx context.Context, msg *Message) (kit.JSONResult, error) {
	msg.callback = make(chan *Message)

	cdp.chReq <- msg

	select {
	case res := <-msg.callback:
		if res.Error != nil {
			return nil, res.Error
		}
		return kit.JSON([]byte(res.Result)), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (cdp *Client) id() uint64 {
	cdp.count++
	return cdp.count
}

func (cdp *Client) close(ctx context.Context, conn *websocket.Conn) {
	<-ctx.Done()
	err := conn.Close()
	if err != nil {
		cdp.chFatal <- err
	}
}
