package cdp

import (
	"context"
	"encoding/json"
	"net/http"
	"sync/atomic"

	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/defaults"
)

// Client is a chrome devtools protocol connection instance.
type Client struct {
	ctx          context.Context
	ctxCancel    func()
	ctxCancelErr error

	wsURL  string
	header http.Header
	ws     Websocketable
	wsConn WebsocketableConn

	callbacks map[uint64]chan *response // buffer for response from chrome

	chReqMsg        chan *requestMsg // request from user
	chRes           chan *response   // response from chrome
	chEvent         chan *Event      // events from chrome
	eventBufferSize int              // size of the chEvent

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
		ctx:             ctx,
		ctxCancel:       cancel,
		callbacks:       map[uint64]chan *response{},
		chReqMsg:        make(chan *requestMsg),
		chRes:           make(chan *response),
		eventBufferSize: 1024,
		wsURL:           websocketURL,
		debug:           defaults.CDP,
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

// EventBuffer set the size of the event buffer, default is 1024
func (cdp *Client) EventBuffer(size int) *Client {
	cdp.eventBufferSize = size
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

// ConnectE to chrome
func (cdp *Client) ConnectE() error {
	cdp.chEvent = make(chan *Event, cdp.eventBufferSize)

	if cdp.ws == nil {
		cdp.ws = DefaultWsClient{}
	}

	conn, err := cdp.ws.Connect(cdp.ctx, cdp.wsURL, cdp.header)
	if err != nil {
		return err
	}

	cdp.wsConn = conn

	go cdp.consumeMsg()

	go cdp.readMsgFromChrome()

	return nil
}

// Connect to chrome
func (cdp *Client) Connect() *Client {
	kit.E(cdp.ConnectE())
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
		case <-cdp.ctx.Done():
			return

		case msg := <-cdp.chReqMsg:
			err := cdp.wsConn.Send(msg.data)
			if err != nil {
				cdp.socketClose(err)
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
	for cdp.ctx.Err() == nil {
		data, err := cdp.wsConn.Read()
		if err != nil {
			cdp.socketClose(err)
			return
		}

		cdp.produceMsg(data)
	}
}

func (cdp *Client) produceMsg(data []byte) {
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

func (cdp *Client) socketClose(err error) {
	cdp.debugLog(err)
	cdp.ctxCancelErr = err
	cdp.ctxCancel()
}
