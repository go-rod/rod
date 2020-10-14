// This file serves for the Page.Evaluate.

package rod

import (
	"errors"
	"fmt"
	"time"

	"github.com/go-rod/rod/lib/assets"
	"github.com/go-rod/rod/lib/assets/js"
	"github.com/go-rod/rod/lib/cdp"
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

	jsHelper bool
}

// Eval creates a EvalOptions with ByValue set to true.
func Eval(js string, args ...interface{}) *EvalOptions {
	return &EvalOptions{true, false, nil, js, args, false, false}
}

// Convert name and jsArgs to Page.Eval, the name is method name in the "lib/assets/helper.js".
func jsHelper(name js.Name, args ...interface{}) *EvalOptions {
	return &EvalOptions{
		ByValue:  true,
		JS:       fmt.Sprintf(`(rod, ...args) => rod.%s.apply(this, args)`, name),
		JSArgs:   args,
		jsHelper: true,
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

// Strings appends each string to JSArgs
func (e *EvalOptions) Strings(list ...string) *EvalOptions {
	for _, s := range list {
		e.JSArgs = append(e.JSArgs, s)
	}
	return e
}

func (e *EvalOptions) formatToJSFunc() string {
	if detectJSFunction(e.JS) {
		return fmt.Sprintf(`function() { return (%s).apply(this, arguments) }`, e.JS)
	}
	return fmt.Sprintf(`function() { return %s }`, e.JS)
}

// Eval is just a shortcut for Page.Evaluate
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

	if opts.jsHelper {
		p.jsCtxLock.Lock()
		id := p.helpers[*p.jsCtxID]
		jsCtx := *p.jsCtxID
		p.jsCtxLock.Unlock()

		if id == "" {
			// inject js helper into the page
			res, err := proto.RuntimeCallFunctionOn{
				ExecutionContextID:  jsCtx,
				FunctionDeclaration: assets.Helper,
			}.Call(p)
			if err != nil {
				return nil, err
			}
			id = res.Result.ObjectID

			p.jsCtxLock.Lock()
			p.helpers[jsCtx] = id
			p.jsCtxLock.Unlock()
		}

		formated = append([]*proto.RuntimeCallArgument{{ObjectID: id}}, formated...)
	}

	return formated, nil
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
		p.helpers = map[proto.RuntimeExecutionContextID]proto.RuntimeRemoteObjectID{}
		*p.jsCtxID = obj.Result.ObjectID.ExecutionID()
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
	p.helpers = map[proto.RuntimeExecutionContextID]proto.RuntimeRemoteObjectID{}
	*p.jsCtxID = obj.Object.ObjectID.ExecutionID()
	p.jsCtxLock.Unlock()
	return nil
}
