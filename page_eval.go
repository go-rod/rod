package rod

import (
	"fmt"
	"time"

	"github.com/go-rod/rod/lib/assets"
	"github.com/go-rod/rod/lib/assets/js"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
)

// Eval options for Page.Evaluate
type Eval struct {
	// If enabled the eval result will be a plain JSON value.
	// If disabled the eval result will be a reference of a remote js object.
	ByValue bool

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

// NewEval options. ByValue will be set to true.
func NewEval(js string, args ...interface{}) *Eval {
	return &Eval{true, nil, js, args, false, false}
}

// This set the obj as ThisObj
func (e *Eval) This(obj *proto.RuntimeRemoteObject) *Eval {
	e.ThisObj = obj
	return e
}

// ByObject disables ByValue.
func (e *Eval) ByObject() *Eval {
	e.ByValue = false
	return e
}

// ByUser enables UserGesture.
func (e *Eval) ByUser() *Eval {
	e.UserGesture = true
	return e
}

// Strings appends each string to JSArgs
func (e *Eval) Strings(list ...string) *Eval {
	for _, s := range list {
		e.JSArgs = append(e.JSArgs, s)
	}
	return e
}

func (e *Eval) formatToJSFunc() string {
	if detectJSFunction(e.JS) {
		return fmt.Sprintf(`function() { return (%s).apply(this, arguments) }`, e.JS)
	}
	return fmt.Sprintf(`function() { return %s }`, e.JS)
}

// We must pass the jsHelper right before we eval it, or the jsHelper may not be generated yet,
// we only inject the js helper on the first Page.EvalWithOption .
func (e *Eval) formatArgs(jsHelper *proto.RuntimeRemoteObject) []*proto.RuntimeCallArgument {
	var jsArgs []interface{}

	if e.jsHelper {
		jsArgs = append([]interface{}{jsHelper}, e.JSArgs...)
	} else {
		jsArgs = e.JSArgs
	}

	formated := []*proto.RuntimeCallArgument{}
	for _, arg := range jsArgs {
		if obj, ok := arg.(*proto.RuntimeRemoteObject); ok { // remote object
			formated = append(formated, &proto.RuntimeCallArgument{ObjectID: obj.ObjectID})
		} else { // plain json data
			formated = append(formated, &proto.RuntimeCallArgument{Value: proto.NewJSON(arg)})
		}
	}
	return formated
}

// Convert name and jsArgs to Page.Eval, the name is method name in the "lib/assets/helper.js".
func jsHelper(name js.Name, args ...interface{}) *Eval {
	return &Eval{
		ByValue:  true,
		JS:       fmt.Sprintf(`(rod, ...args) => rod.%s.apply(this, args)`, name),
		JSArgs:   args,
		jsHelper: true,
	}
}

// Eval js on the page. It's just a shortcut for Page.Evaluate.
func (p *Page) Eval(js string, jsArgs ...interface{}) (*proto.RuntimeRemoteObject, error) {
	return p.Evaluate(NewEval(js, jsArgs...))
}

// Evaluate js on the page.
func (p *Page) Evaluate(opts *Eval) (*proto.RuntimeRemoteObject, error) {
	backoff := utils.BackoffSleeper(30*time.Millisecond, 3*time.Second, nil)
	this := opts.ThisObj
	var err error
	var res *proto.RuntimeCallFunctionOnResult

	// js context will be invalid if a frame is reloaded or not ready, then the isNilContextErr
	// will be true, then we retry the eval again.
	err = utils.Retry(p.ctx, backoff, func() (bool, error) {
		if p.getWindowObj() == nil || opts.ThisObj == nil {
			err := p.initJS(false)
			if err != nil {
				if isNilContextErr(err) {
					return false, nil
				}
				return true, err
			}
		}
		if opts.ThisObj == nil {
			this = p.getWindowObj()
		}

		res, err = proto.RuntimeCallFunctionOn{
			ObjectID:            this.ObjectID,
			AwaitPromise:        true,
			ReturnByValue:       opts.ByValue,
			UserGesture:         opts.UserGesture,
			FunctionDeclaration: opts.formatToJSFunc(),
			Arguments:           opts.formatArgs(p.getJSHelperObj()),
		}.Call(p)
		if opts.ThisObj == nil && isNilContextErr(err) {
			_ = p.initJS(true)
			return false, nil
		}

		return true, err
	})

	if err != nil {
		return nil, err
	}

	if res.ExceptionDetails != nil {
		exp := res.ExceptionDetails.Exception
		return nil, newErr(ErrEval, exp, exp.Description+" "+exp.Value.String())
	}

	return res.Result, nil
}

func (p *Page) initSession() error {
	obj, err := proto.TargetAttachToTarget{
		TargetID: p.TargetID,
		Flatten:  true, // if it's not set no response will return
	}.Call(p)
	if err != nil {
		return err
	}
	p.SessionID = obj.SessionID

	// If we don't enable it, it will cause a lot of unexpected browser behavior.
	// Such as proto.PageAddScriptToEvaluateOnNewDocument won't work.
	p.EnableDomain(&proto.PageEnable{})

	// If we don't enable it, it will remove remote node id whenever we disable the domain
	// even after we re-enable it again we can't query the ids any more.
	p.EnableDomain(&proto.DOMEnable{})

	return nil
}

func (p *Page) initJS(force bool) error {
	contextID, err := p.getExecutionID(force)
	if err != nil {
		return err
	}

	p.jsContextLock.Lock()
	defer p.jsContextLock.Unlock()

	if !force && p.windowObj != nil {
		return nil
	}

	window, err := proto.RuntimeEvaluate{
		Expression: "window",
		ContextID:  contextID,
	}.Call(p)
	if err != nil {
		return err
	}

	helper, err := proto.RuntimeCallFunctionOn{
		ObjectID:            window.Result.ObjectID,
		FunctionDeclaration: assets.Helper,
	}.Call(p)
	if err != nil {
		return err
	}

	p.windowObj = window.Result
	p.jsHelperObj = helper.Result

	return nil
}

// We use this function to make sure every frame(page, iframe) will only have one IsolatedWorld.
func (p *Page) getExecutionID(force bool) (proto.RuntimeExecutionContextID, error) {
	if !p.IsIframe() {
		return 0, nil
	}

	p.jsContextLock.Lock()
	defer p.jsContextLock.Unlock()

	if !force {
		if ctxID, has := p.executionIDs[p.FrameID]; has {
			_, err := proto.RuntimeEvaluate{ContextID: ctxID, Expression: `0`}.Call(p)
			if err == nil {
				return ctxID, nil
			} else if !isNilContextErr(err) {
				return 0, err
			}
		}
	}

	world, err := proto.PageCreateIsolatedWorld{
		FrameID:   p.FrameID,
		WorldName: "rod_iframe_world",
	}.Call(p)
	if err != nil {
		return 0, err
	}

	p.executionIDs[p.FrameID] = world.ExecutionContextID

	return world.ExecutionContextID, nil
}

func (p *Page) getWindowObj() *proto.RuntimeRemoteObject {
	p.jsContextLock.Lock()
	defer p.jsContextLock.Unlock()
	return p.windowObj
}

func (p *Page) getJSHelperObj() *proto.RuntimeRemoteObject {
	p.jsContextLock.Lock()
	defer p.jsContextLock.Unlock()
	return p.jsHelperObj
}
