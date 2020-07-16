package cdp

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/go-rod/rod/lib/defaults"
	"github.com/ysmood/kit"
)

// Client is a devtools protocol connection instance.
type Client struct {
	ctx       context.Context
	ctxCancel func()

	wsURL  string
	header http.Header
	ws     Websocketable
	wsConn WebsocketableConn

	callbacks *sync.Map // buffer for response from browser

	chReq   chan []byte    // request from user
	chRes   chan *response // response from browser
	chEvent chan *Event    // events from browser

	count uint64

	debug    bool
	debugLog func(interface{})
}

// Request to send to browser
type Request struct {
	ID        uint64      `json:"id"`
	SessionID string      `json:"sessionId,omitempty"`
	Method    string      `json:"method"`
	Params    interface{} `json:"params,omitempty"`
}

// Event from browser
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
	// Connect to server
	Connect(ctx context.Context, url string, header http.Header) (WebsocketableConn, error)
}

// WebsocketableConn represents a connection session
type WebsocketableConn interface {
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
func New(websocketURL string) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	cdp := &Client{
		ctx:       ctx,
		ctxCancel: cancel,
		callbacks: &sync.Map{},
		chReq:     make(chan []byte),
		chRes:     make(chan *response),
		chEvent:   make(chan *Event),
		wsURL:     websocketURL,
		debug:     defaults.CDP,
		debugLog:  defaultDebugLog,
	}

	return cdp
}

// Context set the context
func (cdp *Client) Context(ctx context.Context, cancel func()) *Client {
	cdp.ctx = ctx
	cdp.ctxCancel = cancel
	return cdp
}

// Header set the header of the remote control websocket request
func (cdp *Client) Header(header http.Header) *Client {
	cdp.header = header
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

// DebugLog override the defaultDebugLog function
func (cdp *Client) DebugLog(fn func(interface{})) *Client {
	cdp.debugLog = fn
	return cdp
}

// ConnectE to browser
func (cdp *Client) ConnectE() error {
	if cdp.ws == nil {
		cdp.ws = DefaultWsClient{
			WriteBufferSize: 1 * 1024 * 1024,
		}
	}

	conn, err := cdp.ws.Connect(cdp.ctx, cdp.wsURL, cdp.header)
	if err != nil {
		return err
	}

	cdp.wsConn = conn

	go cdp.consumeMsg()

	go cdp.readMsgFromBrowser()

	return nil
}

// Connect to browser
func (cdp *Client) Connect() *Client {
	kit.E(cdp.ConnectE())
	return cdp
}

// Call a method and get its response, if ctx is nil context.Background() will be used
func (cdp *Client) Call(ctx context.Context, sessionID, method string, params interface{}) ([]byte, error) {
	req := &Request{
		ID:        atomic.AddUint64(&cdp.count, 1),
		SessionID: sessionID,
		Method:    method,
		Params:    params,
	}

	if cdp.debug {
		cdp.debugLog(req)
	}

	data, err := json.Marshal(req)
	kit.E(err)

	callback := make(chan *response)

	cdp.callbacks.Store(req.ID, callback)
	defer cdp.callbacks.Delete(req.ID)

	select {
	case <-cdp.ctx.Done():
		return nil, cdp.ctx.Err()

	case <-ctx.Done():
		return nil, ctx.Err()
	case cdp.chReq <- data:
	}

	select {
	case <-cdp.ctx.Done():
		return nil, cdp.ctx.Err()

	case <-ctx.Done():
		return nil, ctx.Err()

	case res := <-callback:
		if res.Error != nil {
			return nil, res.Error
		}
		return res.Result, nil
	}

}

// Event returns a channel that will emit browser devtools protocol events. Must be consumed or will block producer.
func (cdp *Client) Event() <-chan *Event {
	return cdp.chEvent
}

type requestMsg struct {
	request *Request
	data    []byte
}

// consume messages from client and browser
func (cdp *Client) consumeMsg() {
	for {
		select {
		case <-cdp.ctx.Done():
			return

		case data, ok := <-cdp.chReq:
			if !ok {
				return
			}

			err := cdp.wsConn.Send(data)
			if err != nil {
				cdp.close(err)
				return
			}

		case res, ok := <-cdp.chRes:
			if !ok {
				return
			}

			callback, has := cdp.callbacks.Load(res.ID)
			if has {
				select {
				case <-cdp.ctx.Done():
					return
				case callback.(chan *response) <- res:
				}
			}
		}
	}
}

// response from browser
type response struct {
	ID     uint64          `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *Error          `json:"error,omitempty"`
}

func (cdp *Client) readMsgFromBrowser() {
	for cdp.ctx.Err() == nil {
		data, err := cdp.wsConn.Read()
		if err != nil {
			cdp.close(err)
			return
		}

		if kit.JSON(data).Get("id").Exists() {
			var res response
			err := json.Unmarshal(data, &res)
			kit.E(err)
			if cdp.debug {
				cdp.debugLog(&res)
			}
			select {
			case <-cdp.ctx.Done():
				return
			case cdp.chRes <- &res:
			}
		} else {
			var evt Event
			err := json.Unmarshal(data, &evt)
			kit.E(err)
			if cdp.debug {
				cdp.debugLog(&evt)
			}
			select {
			case <-cdp.ctx.Done():
				return
			case cdp.chEvent <- &evt:
			}
		}
	}
}

func (cdp *Client) close(err error) {
	if cdp.debug {
		cdp.debugLog(err)
	}
	cdp.ctxCancel()
}
