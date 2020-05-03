package cdp

import (
	"context"
	"encoding/json"
	"sync/atomic"

	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/defaults"
	"github.com/ysmood/rod/lib/launcher"
)

// Client is a chrome devtools protocol connection instance.
type Client struct {
	ctx          context.Context
	ctxCancel    func()
	ctxCancelErr error

	url string
	ws  Websocketable

	callbacks map[uint64]chan *response
	chReqMsg  chan *requestMsg
	chRes     chan *response
	chEvent   chan *Event

	count uint64

	debug bool
}

// Request to send to chrome
type Request struct {
	ID        uint64      `json:"id"`
	SessionID string      `json:"sessionId,omitempty"`
	Method    string      `json:"method"`
	Params    interface{} `json:"params,omitempty"`
}

// Event from chrome
type Event struct {
	SessionID string          `json:"sessionId,omitempty"`
	Method    string          `json:"method"`
	Params    json.RawMessage `json:"params,omitempty"`
}

// Error of the Response
type Error struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data"`
}

// Websocketable enables you to choose the websocket lib you want to use.
// By default cdp use github.com/gorilla/websocket
type Websocketable interface {
	// Send text message only
	Send([]byte) error
	// Read returns text message only
	Read() ([]byte, error)
}

// Error interface
func (e *Error) Error() string {
	return kit.MustToJSON(e)
}

// New creates a cdp connection, all messages from Client.Event must be received or they will block the client.
func New() *Client {
	ctx, cancel := context.WithCancel(context.Background())

	cdp := &Client{
		ctx:       ctx,
		ctxCancel: cancel,
		callbacks: map[uint64]chan *response{},
		chReqMsg:  make(chan *requestMsg),
		chRes:     make(chan *response),
		chEvent:   make(chan *Event),
		debug:     defaults.CDP,
	}

	return cdp
}

// Context set the context
func (cdp *Client) Context(ctx context.Context) *Client {
	ctx, cancel := context.WithCancel(ctx)
	cdp.ctx = ctx
	cdp.ctxCancel = cancel
	return cdp
}

// URL set the remote control url. The url can be something like http://localhost:9222/* or ws://localhost:9222/*.
// Only the scheme, host, port of the url will be used.
func (cdp *Client) URL(url string) *Client {
	cdp.url = url
	return cdp
}

// Websocket set the websocket lib to use
func (cdp *Client) Websocket(ws Websocketable) *Client {
	cdp.ws = ws
	return cdp
}

// Debug is the flag to enable debug log to stdout.
func (cdp *Client) Debug(enable bool) *Client {
	cdp.debug = enable
	return cdp
}

// Connect to chrome
func (cdp *Client) Connect() *Client {
	if cdp.ws == nil {
		wsURL, err := launcher.GetWebSocketDebuggerURL(cdp.ctx, cdp.url)
		kit.E(err)
		cdp.ws = NewDefaultWsClient(cdp.ctx, wsURL, nil)
	}

	go cdp.consumeMsg()

	go cdp.readMsgFromChrome()

	return cdp
}

// Call a method and get its response, if ctx is nil context.Background() will be used
func (cdp *Client) Call(ctx context.Context, sessionID, method string, params interface{}) (res []byte, err error) {
	req := &Request{
		ID:        atomic.AddUint64(&cdp.count, 1),
		SessionID: sessionID,
		Method:    method,
		Params:    params,
	}

	cdp.debugLog(req)

	data, err := json.Marshal(req)
	kit.E(err)

	callback := make(chan *response)

	cdp.chReqMsg <- &requestMsg{
		request:  req,
		callback: callback,
		data:     data,
	}

	select {
	case data := <-callback:
		if data.Error != nil {
			return nil, data.Error
		}
		return data.Result, nil

	case <-cdp.ctx.Done():
		if cdp.ctxCancelErr != nil {
			err = cdp.ctxCancelErr
		} else {
			err = cdp.ctx.Err()
		}

	case <-ctx.Done():
		err = ctx.Err()

		// to prevent req from leaking
		cdp.chRes <- &response{ID: req.ID}
		<-callback
	}

	return
}

// Event returns a channel that will emit chrome devtools protocol events. Must be consumed or will block producer.
func (cdp *Client) Event() chan *Event {
	return cdp.chEvent
}

type requestMsg struct {
	request  *Request
	data     []byte
	callback chan *response
}

// consume messages from client and chrome
func (cdp *Client) consumeMsg() {
	for {
		select {
		case msg := <-cdp.chReqMsg:
			err := cdp.ws.Send(msg.data)
			if err != nil {
				cdp.close(err)
				return
			}
			cdp.callbacks[msg.request.ID] = msg.callback

		case res := <-cdp.chRes:
			callback, has := cdp.callbacks[res.ID]
			if has {
				delete(cdp.callbacks, res.ID)
				callback <- res
			}
		}
	}
}

// response from chrome
type response struct {
	ID     uint64          `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *Error          `json:"error,omitempty"`
}

func (cdp *Client) readMsgFromChrome() {
	for {
		data, err := cdp.ws.Read()
		if err != nil {
			cdp.close(err)
			return
		}

		if kit.JSON(data).Get("id").Exists() {
			var res response
			err := json.Unmarshal(data, &res)
			kit.E(err)
			cdp.debugLog(&res)
			cdp.chRes <- &res
		} else {
			var evt Event
			err := json.Unmarshal(data, &evt)
			kit.E(err)
			cdp.debugLog(&evt)
			cdp.chEvent <- &evt
		}
	}
}

func (cdp *Client) close(err error) {
	cdp.debugLog(err)
	cdp.ctxCancelErr = err
	cdp.ctxCancel()
}
