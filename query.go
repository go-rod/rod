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
	if els.Empty() {
		return nil
	}
	return els[0]
}

// Last returns the last element, if the list is empty returns nil
func (els Elements) Last() *Element {
	if els.Empty() {
		return nil
	}
	return els[len(els)-1]
}

// Empty returns true if the list is empty
func (els Elements) Empty() bool {
	return len(els) == 0
}

// HasE ...
func (p *Page) HasE(selector string) (bool, error) {
	_, err := p.ElementE(nil, "", selector)
	if IsError(err, ErrElementNotFound) {
		return false, nil
	}
	return err == nil, err
}

// Has an element that matches the css selector
func (p *Page) Has(selector string) bool {
	has, err := p.HasE(selector)
	kit.E(err)
	return has
}

// HasXE ...
func (p *Page) HasXE(selector string) (bool, error) {
	_, err := p.ElementXE(nil, "", selector)
	if IsError(err, ErrElementNotFound) {
		return false, nil
	}
	return err == nil, err
}

// HasX an element that matches the XPath selector
func (p *Page) HasX(selector string) bool {
	has, err := p.HasXE(selector)
	kit.E(err)
	return has
}

// HasMatchesE ...
func (p *Page) HasMatchesE(selector, regex string) (bool, error) {
	_, err := p.ElementMatchesE(nil, "", selector, regex)
	if IsError(err, ErrElementNotFound) {
		return false, nil
	}
	return err == nil, err
}

// HasMatches an element that matches the css selector and its text matches the regex.
func (p *Page) HasMatches(selector, regex string) bool {
	has, err := p.HasMatchesE(selector, regex)
	kit.E(err)
	return has
}

// ElementE finds element by css selector
func (p *Page) ElementE(sleeper kit.Sleeper, objectID, selector string) (*Element, error) {
	return p.ElementByJSE(sleeper, objectID, `s => (this.document || this).querySelector(s)`, []interface{}{selector})
}

// Element retries until returns the first element in the page that matches the CSS selector
func (p *Page) Element(selector string) *Element {
	el, err := p.ElementE(p.Sleeper(), "", selector)
	kit.E(err)
	return el
}

// ElementMatchesE ...
func (p *Page) ElementMatchesE(sleeper kit.Sleeper, objectID, selector, regex string) (*Element, error) {
	return p.ElementByJSE(sleeper, objectID, `(sel, reg) => {
		let r = new RegExp(reg)
		let el = Array.from((this.document || this).querySelectorAll(sel)).find(el => el.textContent.match(r))
		return el || null
	}`, []interface{}{selector, regex})
}

// ElementMatches retries until returns the first element in the page that matches the CSS selector and its text matches the regex.
// The regex is the js regex, not golang's.
func (p *Page) ElementMatches(selector, regex string) *Element {
	el, err := p.ElementMatchesE(p.Sleeper(), "", selector, regex)
	kit.E(err)
	return el
}

// ElementXE finds elements by XPath
func (p *Page) ElementXE(sleeper kit.Sleeper, objectID, xpath string) (*Element, error) {
	js := `xpath => document.evaluate(
		xpath, (this.document || this), null, XPathResult.FIRST_ORDERED_NODE_TYPE
	).singleNodeValue`
	return p.ElementByJSE(sleeper, objectID, js, []interface{}{xpath})
}

// ElementX retries until returns the first element in the page that matches the XPath selector
func (p *Page) ElementX(xpath string) *Element {
	el, err := p.ElementXE(p.Sleeper(), "", xpath)
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
func (p *Page) ElementsE(objectID, selector string) (Elements, error) {
	return p.ElementsByJSE(objectID, `s => (this.document || this).querySelectorAll(s)`, []interface{}{selector})
}

// Elements returns all elements that match the css selector
func (p *Page) Elements(selector string) Elements {
	list, err := p.ElementsE("", selector)
	kit.E(err)
	return list
}

// ElementsXE ...
func (p *Page) ElementsXE(objectID, xpath string) (Elements, error) {
	js := `xpath => {
		let iter = document.evaluate(xpath, (this.document || this), null, XPathResult.ORDERED_NODE_ITERATOR_TYPE)
		let list = []
		let el
		while ((el = iter.iterateNext())) list.push(el)
		return list
	}`
	return p.ElementsByJSE(objectID, js, []interface{}{xpath})
}

// ElementsX returns all elements that match the XPath selector
func (p *Page) ElementsX(xpath string) Elements {
	list, err := p.ElementsXE("", xpath)
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
	defer func() { err = p.ReleaseE(objectID) }()

	list, err := p.CallE("Runtime.getProperties", cdp.Object{
		"objectId":      objectID,
		"ownProperties": true,
	})
	kit.E(err)

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

	return elemList, err
}

// ElementsByJS returns the elements from the return value of the js
func (p *Page) ElementsByJS(js string, params ...interface{}) Elements {
	list, err := p.ElementsByJSE("", js, params)
	kit.E(err)
	return list
}

// ElementE ...
func (el *Element) ElementE(selector string) (*Element, error) {
	return el.page.ElementE(nil, el.ObjectID, selector)
}

// Element returns the first child that matches the css selector
func (el *Element) Element(selector string) *Element {
	el, err := el.ElementE(selector)
	kit.E(err)
	return el
}

// ElementXE ...
func (el *Element) ElementXE(xpath string) (*Element, error) {
	return el.page.ElementXE(nil, el.ObjectID, xpath)
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

// ElementMatchesE ...
func (el *Element) ElementMatchesE(selector, regex string) (*Element, error) {
	return el.page.ElementMatchesE(nil, el.ObjectID, selector, regex)
}

// ElementMatches returns the first element in the page that matches the CSS selector and its text matches the regex.
// The regex is the js regex, not golang's.
func (el *Element) ElementMatches(selector, regex string) *Element {
	el, err := el.ElementMatchesE(selector, regex)
	kit.E(err)
	return el
}

// ElementsE ...
func (el *Element) ElementsE(selector string) (Elements, error) {
	return el.page.ElementsE(el.ObjectID, selector)
}

// Elements returns all elements that match the css selector
func (el *Element) Elements(selector string) Elements {
	list, err := el.ElementsE(selector)
	kit.E(err)
	return list
}

// ElementsXE ...
func (el *Element) ElementsXE(xpath string) (Elements, error) {
	return el.page.ElementsXE(el.ObjectID, xpath)
}

// ElementsX returns all elements that match the XPath selector
func (el *Element) ElementsX(xpath string) Elements {
	list, err := el.ElementsXE(xpath)
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
