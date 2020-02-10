// This file contains all query related code for Page and Element to separate the concerns.

package rod

import (
	"context"

	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/cdp"
)

// Elements provides some helpers to deal with element list
type Elements []*Element

// First returns the first element, if the list is empty returns nil
func (els Elements) First() *Element {
	if len(els) > 0 {
		return els[0]
	}
	return nil
}

// Last returns the last element, if the list is empty returns nil
func (els Elements) Last() *Element {
	l := len(els)
	if l > 0 {
		return els[l-1]
	}
	return nil
}

// ElementE finds element by css selector
func (p *Page) ElementE(selector string, sleeper kit.Sleeper) (*Element, error) {
	return p.ElementByJSE(sleeper, "", `s => document.querySelector(s)`, []interface{}{selector})
}

// Element retries until returns the first element in the page that matches the CSS selector
func (p *Page) Element(selector string) *Element {
	el, err := p.ElementE(selector, p.Sleeper())
	kit.E(err)
	return el
}

// ElementMatches retries until returns the first element in the page that matches the CSS selector and its text matches the regex.
// The regex is the js regex, not golang's.
func (p *Page) ElementMatches(selector, regex string) *Element {
	return p.ElementByJS(`(sel, reg) => {
		let r = new RegExp(reg)
		let el = Array.from(document.querySelectorAll(sel)).find(el => el.textContent.match(r))
		return el || null
	}`, selector, regex)
}

// ElementXE finds elements by XPath
func (p *Page) ElementXE(xpath string, sleeper kit.Sleeper) (*Element, error) {
	js := `xpath => document.evaluate(
		xpath, document, null, XPathResult.FIRST_ORDERED_NODE_TYPE
	).singleNodeValue`
	return p.ElementByJSE(sleeper, "", js, []interface{}{xpath})
}

// ElementX retries until returns the first element in the page that matches the XPath selector
func (p *Page) ElementX(xpath string) *Element {
	el, err := p.ElementXE(xpath, p.Sleeper())
	kit.E(err)
	return el
}

// ElementByJSE returns the element from the return value of the js function.
// sleeper is used to sleep before retry the operation.
// If sleeper is nil, no retry will be performed.
// thisID is the this value of the js function, when thisID is "", the this context will be the "window".
// If the js function returns "null", ElementByJSE will retry, you can use custom sleeper to make it only
// retries once.
func (p *Page) ElementByJSE(sleeper kit.Sleeper, thisID, js string, params []interface{}) (*Element, error) {
	var val kit.JSONResult

	if sleeper == nil {
		sleeper = func(_ context.Context) error {
			return &Error{nil, ErrElementNotFound, js}
		}
	}

	err := kit.Retry(p.ctx, sleeper, func() (bool, error) {
		res, err := p.EvalE(false, thisID, js, params)
		if err != nil {
			return true, err
		}
		v := res.Get("result")
		val = &v

		if val.Get("type").String() == "object" && val.Get("subtype").String() == "null" {
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		return nil, err
	}

	if val.Get("subtype").String() != "node" {
		return nil, &Error{nil, ErrExpectElement, val.Raw}
	}

	return &Element{
		page:     p,
		ctx:      p.ctx,
		ObjectID: val.Get("objectId").String(),
	}, nil
}

// ElementByJS retries until returns the element from the return value of the js function
func (p *Page) ElementByJS(js string, params ...interface{}) *Element {
	el, err := p.ElementByJSE(p.Sleeper(), "", js, params)
	kit.E(err)
	return el
}

// ElementsE ...
func (p *Page) ElementsE(selector string) (Elements, error) {
	return p.ElementsByJSE("", `s => document.querySelectorAll(s)`, []interface{}{selector})
}

// Elements returns all elements that match the css selector
func (p *Page) Elements(selector string) Elements {
	list, err := p.ElementsE(selector)
	kit.E(err)
	return list
}

// ElementsXE ...
func (p *Page) ElementsXE(xpath string) (Elements, error) {
	js := `xpath => {
		let iter = document.evaluate(xpath, document, null, XPathResult.ORDERED_NODE_ITERATOR_TYPE)
		let list = []
		let el
		while ((el = iter.iterateNext())) list.push(el)
		return list
	}`
	return p.ElementsByJSE("", js, []interface{}{xpath})
}

// ElementsX returns all elements that match the XPath selector
func (p *Page) ElementsX(xpath string) Elements {
	list, err := p.ElementsXE(xpath)
	kit.E(err)
	return list
}

// ElementsByJSE is different from ElementByJSE, it doesn't do retry
func (p *Page) ElementsByJSE(thisID, js string, params []interface{}) (Elements, error) {
	res, err := p.EvalE(false, thisID, js, params)
	if err != nil {
		return nil, err
	}
	val := res.Get("result")

	if val.Get("subtype").String() != "array" {
		return nil, &Error{nil, ErrExpectElements, val}
	}

	objectID := val.Get("objectId").String()
	if objectID == "" {
		return Elements{}, nil
	}
	defer p.ReleaseObject(res)

	list, err := p.Call("Runtime.getProperties", cdp.Object{
		"objectId":      objectID,
		"ownProperties": true,
	})
	if err != nil {
		return nil, err
	}

	elemList := Elements{}
	for _, obj := range list.Get("result").Array() {
		name := obj.Get("name").String()
		if name == "__proto__" || name == "length" {
			continue
		}
		val := obj.Get("value")

		if val.Get("subtype").String() != "node" {
			return nil, &Error{nil, ErrExpectElements, val}
		}

		elemList = append(elemList, &Element{
			page:     p,
			ctx:      p.ctx,
			ObjectID: val.Get("objectId").String(),
		})
	}

	return elemList, nil
}

// ElementsByJS returns the elements from the return value of the js
func (p *Page) ElementsByJS(js string, params ...interface{}) Elements {
	list, err := p.ElementsByJSE("", js, params)
	kit.E(err)
	return list
}

// ElementE ...
func (el *Element) ElementE(selector string) (*Element, error) {
	return el.ElementByJSE(`s => this.querySelector(s)`, selector)
}

// Element returns the first child that matches the css selector
func (el *Element) Element(selector string) *Element {
	el, err := el.ElementE(selector)
	kit.E(err)
	return el
}

// ElementXE ...
func (el *Element) ElementXE(xpath string) (*Element, error) {
	js := `xpath => document.evaluate(
		xpath, this, null, XPathResult.FIRST_ORDERED_NODE_TYPE
	).singleNodeValue`
	return el.ElementByJSE(js, xpath)
}

// ElementX returns the first child that matches the XPath selector
func (el *Element) ElementX(xpath string) *Element {
	el, err := el.ElementXE(xpath)
	kit.E(err)
	return el
}

// ElementByJSE ...
func (el *Element) ElementByJSE(js string, params ...interface{}) (*Element, error) {
	return el.page.ElementByJSE(nil, el.ObjectID, js, params)
}

// ElementByJS returns the element from the return value of the js
func (el *Element) ElementByJS(js string, params ...interface{}) *Element {
	el, err := el.ElementByJSE(js, params...)
	kit.E(err)
	return el
}

// ParentE ...
func (el *Element) ParentE() (*Element, error) {
	return el.ElementByJSE(`() => this.parentElement`)
}

// Parent returns the parent element
func (el *Element) Parent() *Element {
	parent, err := el.ParentE()
	kit.E(err)
	return parent
}

// NextE ...
func (el *Element) NextE() (*Element, error) {
	return el.ElementByJSE(`() => this.nextElementSibling`)
}

// Next returns the next sibling element
func (el *Element) Next() *Element {
	parent, err := el.NextE()
	kit.E(err)
	return parent
}

// PreviousE ...
func (el *Element) PreviousE() (*Element, error) {
	return el.ElementByJSE(`() => this.previousElementSibling`)
}

// Previous returns the previous sibling element
func (el *Element) Previous() *Element {
	parent, err := el.PreviousE()
	kit.E(err)
	return parent
}

// ElementMatches returns the first element in the page that matches the CSS selector and its text matches the regex.
// The regex is the js regex, not golang's.
func (el *Element) ElementMatches(selector, regex string) *Element {
	return el.ElementByJS(`(sel, reg) => {
		let r = new RegExp(reg)
		let el = Array.from(this.querySelectorAll(sel)).find(el => el.textContent.match(r))
		return el || null
	}`, selector, regex)
}

// ElementsE ...
func (el *Element) ElementsE(selector string) (Elements, error) {
	return el.ElementsByJSE(`s => this.querySelectorAll(s)`, selector)
}

// Elements returns all elements that match the css selector
func (el *Element) Elements(selector string) Elements {
	list, err := el.ElementsE(selector)
	kit.E(err)
	return list
}

// ElementsXE ...
func (el *Element) ElementsXE(selector string) (Elements, error) {
	js := `xpath => {
		let iter = document.evaluate(xpath, this, null, XPathResult.ORDERED_NODE_ITERATOR_TYPE)
		let list = []
		let el
		while ((el = iter.iterateNext())) list.push(el)
		return list
	}`
	return el.ElementsByJSE(js, selector)
}

// ElementsX returns all elements that match the XPath selector
func (el *Element) ElementsX(selector string) Elements {
	list, err := el.ElementsXE(selector)
	kit.E(err)
	return list
}

// ElementsByJSE ...
func (el *Element) ElementsByJSE(js string, params ...interface{}) (Elements, error) {
	return el.page.ElementsByJSE(el.ObjectID, js, params)
}

// ElementsByJS returns the elements from the return value of the js
func (el *Element) ElementsByJS(js string, params ...interface{}) Elements {
	list, err := el.ElementsByJSE(js, params...)
	kit.E(err)
	return list
}
