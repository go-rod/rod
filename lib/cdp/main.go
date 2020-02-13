package cdp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"

	"github.com/gorilla/websocket"
	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/launcher"
)

// Client is a chrome devtools protocol connection instance.
// To enable debug log, set env "debug_cdp=true".
type Client struct {
	requests map[uint64]*Request
	chReq    chan *Request
	chRes    chan *Response
	chEvent  chan *Event
	count    uint64
}

// Request to send to chrome
type Request struct {
	ID        uint64      `json:"id"`
	SessionID string      `json:"sessionId,omitempty"`
	Method    string      `json:"method"`
	Params    interface{} `json:"params,omitempty"`

	callback chan *Response
}

// Response from chrome
type Response struct {
	ID     uint64 `json:"id"`
	Result *JSON  `json:"result,omitempty"`
	Error  *Error `json:"error,omitempty"`
}

// Error of the Response
type Error struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data"`
}

// Error interface
func (e *Error) Error() string {
	return kit.MustToJSON(e)
}

// Event from chrome
type Event struct {
	SessionID string `json:"sessionId,omitempty"`
	Method    string `json:"method"`
	Params    *JSON  `json:"params,omitempty"`
}

// JSON helper
type JSON struct {
	kit.JSONResult
}

// UnmarshalJSON interface
func (j *JSON) UnmarshalJSON(data []byte) error {
	j.JSONResult = kit.JSON(data)
	return nil
}

// Object is the json object
type Object map[string]interface{}

// Array is the json array
type Array []interface{}

// New creates a cdp connection, the url should be something like http://localhost:9222.
// All messages from Client.Event must be received or they will block the client.
func New(ctx context.Context, url string) (*Client, error) {
	cdp := &Client{
		requests: map[uint64]*Request{},
		chReq:    make(chan *Request),
		chRes:    make(chan *Response),
		chEvent:  make(chan *Event),
	}

	wsURL, err := launcher.GetWebSocketDebuggerURL(url)
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
		case req := <-cdp.chReq:
			req.ID = cdp.id()
			data, err := json.Marshal(req)
			checkPanic(err)
			debug(req)
			err = conn.WriteMessage(websocket.TextMessage, data)
			checkPanic(err)
			cdp.requests[req.ID] = req

		case res := <-cdp.chRes:
			req := cdp.requests[res.ID]
			delete(cdp.requests, res.ID)
			req.callback <- res
		}
	}
}

func (cdp *Client) handleRes(ctx context.Context, conn *websocket.Conn) {
	for ctx.Err() == nil {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			var netErr *net.OpError
			notClosed := errors.As(err, &netErr) &&
				netErr.Err.Error() != "use of closed network connection"
			if err != io.EOF && notClosed {
				checkPanic(err)
			}
			return
		}

		if msgType == websocket.TextMessage {
			if kit.JSON(data).Get("id").Exists() {
				var res Response
				err = json.Unmarshal(data, &res)
				checkPanic(err)
				debug(&res)
				cdp.chRes <- &res
			} else {
				var e Event
				err = json.Unmarshal(data, &e)
				debug(&e)
				cdp.chEvent <- &e
			}
		}
	}
}

// Event will emit chrome devtools protocol events
func (cdp *Client) Event() chan *Event {
	return cdp.chEvent
}

// Call a method and get its response
func (cdp *Client) Call(ctx context.Context, req *Request) (kit.JSONResult, error) {
	req.callback = make(chan *Response)

	cdp.chReq <- req

	select {
	case res := <-req.callback:
		if res.Error != nil {
			return nil, res.Error
		}
		return res.Result.JSONResult, nil
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
		checkPanic(err)
	}
}
