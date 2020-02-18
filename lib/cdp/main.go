package cdp

import (
	"context"
	"encoding/json"

	"github.com/gorilla/websocket"
	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/launcher"
)

// Client is a chrome devtools protocol connection instance.
// To enable debug log, set env "debug_cdp=true".
type Client struct {
	ctx  context.Context
	url  string
	conn *websocket.Conn

	requests map[uint64]*Request
	chReq    chan *Request
	chRes    chan *Response
	chEvent  chan *Event

	count  uint64
	cancel func()

	readBufferSize  int
	writeBufferSize int
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
func New(url string) *Client {
	cdp := &Client{
		ctx:      context.Background(),
		url:      url,
		requests: map[uint64]*Request{},
		chReq:    make(chan *Request),
		chRes:    make(chan *Response),
		chEvent:  make(chan *Event),
		cancel:   func() {},
	}

	return cdp
}

// Context set the context
func (cdp *Client) Context(ctx context.Context) *Client {
	cdp.ctx = ctx
	return cdp
}

// Cancel set the cancel callback
func (cdp *Client) Cancel(fn func()) *Client {
	cdp.cancel = fn
	return cdp
}

// Buffer set the read and write buffer for websocket
func (cdp *Client) Buffer(read, write int) *Client {
	cdp.readBufferSize = read
	cdp.writeBufferSize = write
	return cdp
}

// Connect to chrome
func (cdp *Client) Connect() *Client {
	wsURL, err := launcher.GetWebSocketDebuggerURL(cdp.url)
	kit.E(err)

	dialer := *websocket.DefaultDialer
	dialer.ReadBufferSize = cdp.readBufferSize
	dialer.WriteBufferSize = cdp.writeBufferSize

	if dialer.ReadBufferSize == 0 {
		dialer.ReadBufferSize = 25 * 1024 * 1024
	}
	if dialer.WriteBufferSize == 0 {
		dialer.WriteBufferSize = 10 * 1024 * 1024
	}

	conn, _, err := dialer.DialContext(cdp.ctx, wsURL, nil)
	kit.E(err)

	cdp.conn = conn

	go cdp.close()

	go cdp.handleReq()

	go cdp.handleRes()

	return cdp
}

func (cdp *Client) handleReq() {
	for {
		select {
		case <-cdp.ctx.Done():
			return

		case req := <-cdp.chReq:
			req.ID = cdp.id()
			data, err := json.Marshal(req)
			checkPanic(err)
			debug(req)
			err = cdp.conn.WriteMessage(websocket.TextMessage, data)
			checkPanic(err)
			cdp.requests[req.ID] = req

		case res := <-cdp.chRes:
			req, has := cdp.requests[res.ID]
			if has {
				delete(cdp.requests, res.ID)
				req.callback <- res
			}
		}
	}
}

func (cdp *Client) handleRes() {
	for cdp.ctx.Err() == nil {
		msgType, data, err := cdp.conn.ReadMessage()
		if err != nil {
			debug(err)
			cdp.cancel()
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
	case <-ctx.Done():
		// to prevent req from leaking
		cdp.chRes <- &Response{ID: req.ID}
		<-req.callback

		return nil, ctx.Err()

	case res := <-req.callback:
		if res.Error != nil {
			return nil, res.Error
		}
		return res.Result.JSONResult, nil
	}
}

func (cdp *Client) id() uint64 {
	cdp.count++
	return cdp.count
}

func (cdp *Client) close() {
	<-cdp.ctx.Done()
	err := cdp.conn.Close()
	if !isClosedErr(err) {
		checkPanic(err)
	}
}
