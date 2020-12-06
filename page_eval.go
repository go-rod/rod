// This file serves for the Page.Evaluate.

package rod

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod/lib/cdp"
	"github.com/go-rod/rod/lib/js"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/gson"
)

// EvalOptions for Page.Evaluate
type EvalOptions struct {
	// If enabled the eval result will be a plain JSON value.
	// If disabled the eval result will be a reference of a remote js object.
	ByValue bool

	AwaitPromise bool

	// ThisObj represents the "this" object in the JS
	ThisObj *proto.RuntimeRemoteObject

	// JS code to eval
	JS string

	// JSArgs represents the arguments in the JS if the JS is a function definition.
	// If an argument is *proto.RuntimeRemoteObject type, the corresponding remote object will be used.
	// Or it will be passed as a plain JSON value.
	JSArgs []interface{}

	// Whether execution should be treated as initiated by user in the UI.
	UserGesture bool

	jsHelper *js.Function
}

// Eval creates a EvalOptions with ByValue set to true.
func Eval(js string, args ...interface{}) *EvalOptions {
	return &EvalOptions{
		ByValue:      true,
		AwaitPromise: false,
		ThisObj:      nil,
		JS:           js,
		JSArgs:       args,
		UserGesture:  false,
		jsHelper:     nil,
	}
}

// EvalHelper creates a special EvalOptions that will cache the fn on the page js context.
// Useful when you want to extend the helpers of Rod, such as create your own selector helpers.
func EvalHelper(fn *js.Function, args ...interface{}) *EvalOptions {
	return &EvalOptions{
		ByValue:  true,
		JSArgs:   args,
		JS:       `({fn}, ...args) => fn.apply(this, args)`,
		jsHelper: fn,
	}
}

// This set the obj as ThisObj
func (e *EvalOptions) This(obj *proto.RuntimeRemoteObject) *EvalOptions {
	e.ThisObj = obj
	return e
}

// ByObject disables ByValue.
func (e *EvalOptions) ByObject() *EvalOptions {
	e.ByValue = false
	return e
}

// ByUser enables UserGesture.
func (e *EvalOptions) ByUser() *EvalOptions {
	e.UserGesture = true
	return e
}

// ByPromise enables AwaitPromise.
func (e *EvalOptions) ByPromise() *EvalOptions {
	e.AwaitPromise = true
	return e
}

func (e *EvalOptions) formatToJSFunc() string {
	js := strings.TrimSpace(e.JS)
	if detectJSFunction(js) {
		return fmt.Sprintf(`function() { return (%s).apply(this, arguments) }`, js)
	}
	return fmt.Sprintf(`function() { return %s }`, js)
}

// Eval is just a shortcut for Page.Evaluate with AwaitPromise set true.
func (p *Page) Eval(js string, jsArgs ...interface{}) (*proto.RuntimeRemoteObject, error) {
	return p.Evaluate(Eval(js, jsArgs...).ByPromise())
}

// Evaluate js on the page.
func (p *Page) Evaluate(opts *EvalOptions) (res *proto.RuntimeRemoteObject, err error) {
	var backoff utils.Sleeper

	// js context will be invalid if a frame is reloaded or not ready, then the isNilContextErr
	// will be true, then we retry the eval again.
	for {
		res, err = p.evaluate(opts)
		if err != nil && errors.Is(err, cdp.ErrCtxNotFound) {
			if opts.ThisObj != nil {
				return nil, &ErrObjectNotFound{opts.ThisObj}
			}

			if backoff == nil {
				backoff = utils.BackoffSleeper(30*time.Millisecond, 3*time.Second, nil)
			} else {
				_ = backoff(p.ctx)
			}

			err := p.updateJSCtxID()
			if err != nil {
				return nil, err
			}

			continue
		}
		return
	}
}

func (p *Page) evaluate(opts *EvalOptions) (*proto.RuntimeRemoteObject, error) {
	args, err := p.formatArgs(opts)
	if err != nil {
		return nil, err
	}

	req := proto.RuntimeCallFunctionOn{
		AwaitPromise:        opts.AwaitPromise,
		ReturnByValue:       opts.ByValue,
		UserGesture:         opts.UserGesture,
		FunctionDeclaration: opts.formatToJSFunc(),
		Arguments:           args,
	}

	if opts.ThisObj == nil {
		req.ExecutionContextID = p.getJSCtxID()
	} else {
		req.ObjectID = opts.ThisObj.ObjectID
	}

	res, err := req.Call(p)
	if err != nil {
		return nil, err
	}

	if res.ExceptionDetails != nil {
		return nil, &ErrEval{res.ExceptionDetails}
	}

	return res.Result, nil
}

// Expose fn to the page's window object with the name. The exposure survives reloads.
// Call stop to unbind the fn.
func (p *Page) Expose(name string, fn func(gson.JSON) (interface{}, error)) (stop func() error, err error) {
	bind := "_" + utils.RandString(8)

	err = proto.RuntimeAddBinding{Name: bind, ExecutionContextID: p.getJSCtxID()}.Call(p)
	if err != nil {
		return
	}

	code := fmt.Sprintf(`(%s)("%s", "%s")`, js.ExposeFunc.Definition, name, bind)

	_, err = p.Evaluate(Eval(code))
	if err != nil {
		return
	}

	remove, err := p.EvalOnNewDocument(code)
	if err != nil {
		return
	}

	p, cancel := p.WithCancel()

	stop = func() error {
		defer cancel()
		err := remove()
		if err != nil {
			return err
		}
		return proto.RuntimeRemoveBinding{Name: bind}.Call(p)
	}

	go p.EachEvent(func(e *proto.RuntimeBindingCalled) {
		if e.Name == bind {
			payload := gson.NewFrom(e.Payload)
			res, err := fn(payload.Get("req"))
			code := fmt.Sprintf("(res, err) => %s(res, err)", payload.Get("cb").Str())
			_, _ = p.Evaluate(Eval(code, res, err))
		}
	})()

	return
}

func (p *Page) initSession() error {
	session, err := proto.TargetAttachToTarget{
		TargetID: p.TargetID,
		Flatten:  true, // if it's not set no response will return
	}.Call(p)
	if err != nil {
		return err
	}
	p.SessionID = session.SessionID

	// If we don't enable it, it will cause a lot of unexpected browser behavior.
	// Such as proto.PageAddScriptToEvaluateOnNewDocument won't work.
	p.EnableDomain(&proto.PageEnable{})

	// If we don't enable it, it will remove remote node id whenever we disable the domain
	// even after we re-enable it again we can't query the ids any more.
	p.EnableDomain(&proto.DOMEnable{})

	p.FrameID = proto.PageFrameID(p.TargetID)

	return p.updateJSCtxID()
}

func (p *Page) formatArgs(opts *EvalOptions) ([]*proto.RuntimeCallArgument, error) {
	formated := []*proto.RuntimeCallArgument{}
	for _, arg := range opts.JSArgs {
		if obj, ok := arg.(*proto.RuntimeRemoteObject); ok { // remote object
			formated = append(formated, &proto.RuntimeCallArgument{ObjectID: obj.ObjectID})
		} else { // plain json data
			formated = append(formated, &proto.RuntimeCallArgument{Value: gson.New(arg)})
		}
	}

	if opts.jsHelper != nil {
		p.jsCtxLock.Lock()
		id, err := p.ensureJSHelper(opts.jsHelper)
		p.jsCtxLock.Unlock()
		if err != nil {
			return nil, err
		}

		formated = append([]*proto.RuntimeCallArgument{{ObjectID: id}}, formated...)
	}

	return formated, nil
}

func (p *Page) ensureJSHelper(fn *js.Function) (proto.RuntimeRemoteObjectID, error) {
	if p.helpers == nil {
		p.helpers = map[proto.RuntimeExecutionContextID]map[string]proto.RuntimeRemoteObjectID{}
	}

	list, ok := p.helpers[*p.jsCtxID]
	if !ok {
		list = map[string]proto.RuntimeRemoteObjectID{}
		p.helpers[*p.jsCtxID] = list
	}

	fns, has := list[js.Functions.Name]
	if !has {
		res, err := proto.RuntimeCallFunctionOn{
			ExecutionContextID:  *p.jsCtxID,
			FunctionDeclaration: js.Functions.Definition,
		}.Call(p)
		if err != nil {
			return "", err
		}
		fns = res.Result.ObjectID
		list[js.Functions.Name] = fns
	}

	id, has := list[fn.Name]
	if !has {
		for _, dep := range fn.Dependencies {
			_, err := p.ensureJSHelper(dep)
			if err != nil {
				return "", err
			}
		}

		res, err := proto.RuntimeCallFunctionOn{
			ExecutionContextID: *p.jsCtxID,
			Arguments:          []*proto.RuntimeCallArgument{{ObjectID: fns}},

			FunctionDeclaration: fmt.Sprintf(
				// We wrap an extra {fn: fn} here to reduce the response body size,
				// we only need the object id, but the cdp will return the whole function string.
				"functions => { functions.%s = %s; return { fn: functions.%s } }",
				fn.Name, fn.Definition, fn.Name,
			),
		}.Call(p)
		if err != nil {
			return "", err
		}

		id = res.Result.ObjectID
		list[fn.Name] = id
	}

	return id, nil
}

func (p *Page) getJSCtxID() proto.RuntimeExecutionContextID {
	p.jsCtxLock.Lock()
	defer p.jsCtxLock.Unlock()
	return *p.jsCtxID
}

func (p *Page) updateJSCtxID() error {
	if !p.IsIframe() {
		obj, err := proto.RuntimeEvaluate{Expression: "window"}.Call(p)
		if err != nil {
			return err
		}

		p.jsCtxLock.Lock()
		*p.jsCtxID = obj.Result.ObjectID.ExecutionID()
		p.helpers = nil
		p.jsCtxLock.Unlock()
		return nil
	}

	owner, err := proto.DOMGetFrameOwner{FrameID: p.FrameID}.Call(p)
	if err != nil {
		return err
	}

	node, err := proto.DOMDescribeNode{BackendNodeID: owner.BackendNodeID, Pierce: true}.Call(p)
	if err != nil {
		return err
	}

	obj, err := proto.DOMResolveNode{BackendNodeID: node.Node.ContentDocument.BackendNodeID}.Call(p)
	if err != nil {
		return err
	}

	p.jsCtxLock.Lock()
	delete(p.helpers, *p.jsCtxID)
	*p.jsCtxID = obj.Object.ObjectID.ExecutionID()
	p.jsCtxLock.Unlock()
	return nil
}
