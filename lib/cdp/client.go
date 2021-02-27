// Package cdp for application layer communication with browser.
package cdp

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/go-rod/rod/lib/defaults"
	"github.com/go-rod/rod/lib/utils"
)

// Client is a devtools protocol connection instance.
type Client struct {
	ctx   context.Context
	close func()

	wsURL  string
	header http.Header
	ws     WebSocketable

	callbacks *sync.Map // buffer for response from browser

	chReq   chan []byte    // request from user
	chRes   chan *Response // response from browser
	chEvent chan *Event    // events from browser

	count uint64

	logger utils.Logger
}

// Request to send to browser
type Request struct {
	ID        int         `json:"id"`
	SessionID string      `json:"sessionId,omitempty"`
	Method    string      `json:"method"`
	Params    interface{} `json:"params,omitempty"`
}

// Response from browser
type Response struct {
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *Error          `json:"error,omitempty"`
}

// Event from browser
type Event struct {
	SessionID string          `json:"sessionId,omitempty"`
	Method    string          `json:"method"`
	Params    json.RawMessage `json:"params,omitempty"`
}

// WebSocketable enables you to choose the websocket lib you want to use.
// Such as you can easily wrap gorilla/websocket and use it as the transport layer.
type WebSocketable interface {
	// Connect to server
	Connect(ctx context.Context, url string, header http.Header) error
	// Send text message only
	Send([]byte) error
	// Read returns text message only
	Read() ([]byte, error)
}

// New creates a cdp connection, all messages from Client.Event must be received or they will block the client.
func New(websocketURL string) *Client {
	return &Client{
		callbacks: &sync.Map{},
		chReq:     make(chan []byte),
		chRes:     make(chan *Response),
		chEvent:   make(chan *Event),
		wsURL:     websocketURL,
		logger:    defaults.CDP,
	}
}

// Header set the header of the remote control websocket request
func (cdp *Client) Header(header http.Header) *Client {
	cdp.header = header
	return cdp
}

// Websocket set the websocket lib to use
func (cdp *Client) Websocket(ws WebSocketable) *Client {
	cdp.ws = ws
	return cdp
}

// Logger sets the logger to log all the requests, responses, and events transferred between Rod and the browser.
// The default format for each type is in file format.go
func (cdp *Client) Logger(l utils.Logger) *Client {
	cdp.logger = l
	return cdp
}

// Connect to browser
func (cdp *Client) Connect(ctx context.Context) error {
	if cdp.ws == nil {
		cdp.ws = &WebSocket{}
	}

	err := cdp.ws.Connect(ctx, cdp.wsURL, cdp.header)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)

	cdp.ctx = ctx
	cdp.close = cancel

	go cdp.consumeMsg()

	go cdp.readMsgFromBrowser()

	return nil
}

// MustConnect is similar to Connect
func (cdp *Client) MustConnect(ctx context.Context) *Client {
	utils.E(cdp.Connect(ctx))
	return cdp
}

// Call a method and get its response, if ctx is nil context.Background() will be used
func (cdp *Client) Call(ctx context.Context, sessionID, method string, params interface{}) ([]byte, error) {
	req := &Request{
		ID:        int(atomic.AddUint64(&cdp.count, 1)),
		SessionID: sessionID,
		Method:    method,
		Params:    params,
	}

	cdp.logger.Println(req)

	data, err := json.Marshal(req)
	utils.E(err)

	callback := make(chan *Response)

	cdp.callbacks.Store(req.ID, callback)
	defer cdp.callbacks.Delete(req.ID)

	select {
	case <-cdp.ctx.Done():
		return nil, &errConnClosed{cdp.ctx.Err()}

	case <-ctx.Done():
		return nil, ctx.Err()

	case cdp.chReq <- data:
	}

	select {
	case <-cdp.ctx.Done():
		return nil, &errConnClosed{cdp.ctx.Err()}

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

// consume messages from client and browser
func (cdp *Client) consumeMsg() {
	for {
		select {
		case <-cdp.ctx.Done():
			return

		case data := <-cdp.chReq:
			err := cdp.ws.Send(data)
			if err != nil {
				cdp.wsClose(err)
				return
			}

		case res := <-cdp.chRes:
			callback, has := cdp.callbacks.Load(res.ID)
			if has {
				select {
				case <-cdp.ctx.Done():
					return
				case callback.(chan *Response) <- res:
				}
			}
		}
	}
}

func (cdp *Client) readMsgFromBrowser() {
	defer close(cdp.chEvent)

	for {
		data, err := cdp.ws.Read()
		if err != nil {
			cdp.wsClose(err)
			return
		}

		var id struct {
			ID int `json:"id"`
		}
		err = json.Unmarshal(data, &id)
		utils.E(err)

		if id.ID != 0 {
			var res Response
			err := json.Unmarshal(data, &res)
			utils.E(err)
			cdp.logger.Println(&res)
			select {
			case <-cdp.ctx.Done():
				return
			case cdp.chRes <- &res:
			}
		} else {
			var evt Event
			err := json.Unmarshal(data, &evt)
			utils.E(err)
			cdp.logger.Println(&evt)
			select {
			case <-cdp.ctx.Done():
				return
			case cdp.chEvent <- &evt:
			}
		}
	}
}

func (cdp *Client) wsClose(err error) {
	cdp.logger.Println(err)
	cdp.close()
}
