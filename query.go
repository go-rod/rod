// This file contains all query related code for Page and Element to separate the concerns.

package rod

import (
	"context"
	"regexp"

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

// Pages provides some helpers to deal with page list
type Pages []*Page

// Find the page that has the specified element with the css selector
func (ps Pages) Find(selector string) *Page {
	for _, page := range ps {
		if page.WaitLoad().Has(selector) {
			return page
		}
	}
	return nil
}

// FindByURL returns the page that has the url that matches the regex
func (ps Pages) FindByURL(regex string) *Page {
	for _, page := range ps {
		url := page.Eval(`() => location.href`).String()
		if regexp.MustCompile(regex).MatchString(url) {
			return page
		}
	}
	return nil
}

// HasE doc is the same as the method Has
func (p *Page) HasE(selector string) (bool, error) {
	_, err := p.ElementE(nil, "", selector)
	if IsError(err, ErrElementNotFound) {
		return false, nil
	}
	return err == nil, err
}

// HasXE doc is the same as the method HasX
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

// HasMatchesE doc is the same as the method HasMatches
func (p *Page) HasMatchesE(selector, regex string) (bool, error) {
	_, err := p.ElementMatchesE(nil, "", selector, regex)
	if IsError(err, ErrElementNotFound) {
		return false, nil
	}
	return err == nil, err
}

// ElementE finds element by css selector
func (p *Page) ElementE(sleeper kit.Sleeper, objectID, selector string) (*Element, error) {
	return p.ElementByJSE(sleeper, objectID, p.jsFn("element"), cdp.Array{selector})
}

// ElementMatchesE doc is the same as the method ElementMatches
func (p *Page) ElementMatchesE(sleeper kit.Sleeper, objectID, selector, regex string) (*Element, error) {
	return p.ElementByJSE(sleeper, objectID, p.jsFn("elementMatches"), cdp.Array{selector, regex})
}

// ElementXE finds elements by XPath
func (p *Page) ElementXE(sleeper kit.Sleeper, objectID, xpath string) (*Element, error) {
	return p.ElementByJSE(sleeper, objectID, p.jsFn("elementX"), cdp.Array{xpath})
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
func (p *Page) ElementByJSE(sleeper kit.Sleeper, thisID, js string, params cdp.Array) (*Element, error) {
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

// ElementsE doc is the same as the method Elements
func (p *Page) ElementsE(objectID, selector string) (Elements, error) {
	return p.ElementsByJSE(objectID, p.jsFn("elements"), cdp.Array{selector})
}

// ElementsXE doc is the same as the method ElementsX
func (p *Page) ElementsXE(objectID, xpath string) (Elements, error) {
	return p.ElementsByJSE(objectID, p.jsFn("elementsX"), cdp.Array{xpath})
}

// ElementsByJSE is different from ElementByJSE, it doesn't do retry
func (p *Page) ElementsByJSE(thisID, js string, params cdp.Array) (Elements, error) {
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

// ElementE doc is the same as the method Element
func (el *Element) ElementE(selector string) (*Element, error) {
	return el.page.ElementE(nil, el.ObjectID, selector)
}

// ElementXE doc is the same as the method ElementX
func (el *Element) ElementXE(xpath string) (*Element, error) {
	return el.page.ElementXE(nil, el.ObjectID, xpath)
}

// ElementByJSE doc is the same as the method ElementByJS
func (el *Element) ElementByJSE(js string, params cdp.Array) (*Element, error) {
	return el.page.ElementByJSE(nil, el.ObjectID, js, params)
}

// ParentE doc is the same as the method Parent
func (el *Element) ParentE() (*Element, error) {
	return el.ElementByJSE(`() => this.parentElement`, nil)
}

// NextE doc is the same as the method Next
func (el *Element) NextE() (*Element, error) {
	return el.ElementByJSE(`() => this.nextElementSibling`, nil)
}

// PreviousE doc is the same as the method Previous
func (el *Element) PreviousE() (*Element, error) {
	return el.ElementByJSE(`() => this.previousElementSibling`, nil)
}

// ElementMatchesE doc is the same as the method ElementMatches
func (el *Element) ElementMatchesE(selector, regex string) (*Element, error) {
	return el.page.ElementMatchesE(nil, el.ObjectID, selector, regex)
}

// ElementsE doc is the same as the method Elements
func (el *Element) ElementsE(selector string) (Elements, error) {
	return el.page.ElementsE(el.ObjectID, selector)
}

// ElementsXE doc is the same as the method ElementsX
func (el *Element) ElementsXE(xpath string) (Elements, error) {
	return el.page.ElementsXE(el.ObjectID, xpath)
}

// ElementsByJSE doc is the same as the method ElementsByJS
func (el *Element) ElementsByJSE(js string, params cdp.Array) (Elements, error) {
	return el.page.ElementsByJSE(el.ObjectID, js, params)
}
