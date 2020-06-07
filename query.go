// This file contains all query related code for Page and Element to separate the concerns.

package rod

import (
	"context"
	"regexp"

	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/proto"
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
		if page.Has(selector) {
			return page
		}
	}
	return nil
}

// FindByURLE returns the page that has the url that matches the regex
func (ps Pages) FindByURLE(regex string) (*Page, error) {
	for _, page := range ps {
		res, err := page.EvalE(true, "", `() => location.href`, nil)
		if err != nil {
			return nil, err
		}
		url := res.Value.String()
		if regexp.MustCompile(regex).MatchString(url) {
			return page, nil
		}
	}
	return nil, nil
}

// HasE doc is similar to the method Has
func (p *Page) HasE(selector string) (bool, error) {
	_, err := p.ElementE(nil, "", selector)
	if IsError(err, ErrElementNotFound) {
		return false, nil
	}
	return err == nil, err
}

// HasXE doc is similar to the method HasX
func (p *Page) HasXE(selector string) (bool, error) {
	_, err := p.ElementXE(nil, "", selector)
	if IsError(err, ErrElementNotFound) {
		return false, nil
	}
	return err == nil, err
}

// HasMatchesE doc is similar to the method HasMatches
func (p *Page) HasMatchesE(selector, regex string) (bool, error) {
	_, err := p.ElementMatchesE(nil, "", selector, regex)
	if IsError(err, ErrElementNotFound) {
		return false, nil
	}
	return err == nil, err
}

// ElementE finds element by css selector
func (p *Page) ElementE(sleeper kit.Sleeper, objectID proto.RuntimeRemoteObjectID, selector string) (*Element, error) {
	return p.ElementByJSE(sleeper, objectID, p.jsFn("element"), Array{selector})
}

// ElementMatchesE doc is similar to the method ElementMatches
func (p *Page) ElementMatchesE(sleeper kit.Sleeper, objectID proto.RuntimeRemoteObjectID, selector, regex string) (*Element, error) {
	return p.ElementByJSE(sleeper, objectID, p.jsFn("elementMatches"), Array{selector, regex})
}

// ElementXE finds elements by XPath
func (p *Page) ElementXE(sleeper kit.Sleeper, objectID proto.RuntimeRemoteObjectID, xpath string) (*Element, error) {
	return p.ElementByJSE(sleeper, objectID, p.jsFn("elementX"), Array{xpath})
}

// ElementByJSE returns the element from the return value of the js function.
// sleeper is used to sleep before retry the operation.
// If sleeper is nil, no retry will be performed.
// thisID is the this value of the js function, when thisID is "", the this context will be the "window".
// If the js function returns "null", ElementByJSE will retry, you can use custom sleeper to make it only
// retries once.
func (p *Page) ElementByJSE(sleeper kit.Sleeper, thisID proto.RuntimeRemoteObjectID, js string, params Array) (*Element, error) {
	var res *proto.RuntimeRemoteObject
	var err error

	if sleeper == nil {
		sleeper = func(_ context.Context) error {
			return &Error{nil, ErrElementNotFound, js}
		}
	}

	if p.browser.trace {
		defer p.traceFn(js, params)()
	}

	err = kit.Retry(p.ctx, sleeper, func() (bool, error) {
		res, err = p.EvalE(false, thisID, js, params)
		if err != nil {
			return true, err
		}

		if res.Type == proto.RuntimeRemoteObjectTypeObject && res.Subtype == proto.RuntimeRemoteObjectSubtypeNull {
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		return nil, err
	}

	if res.Subtype != proto.RuntimeRemoteObjectSubtypeNode {
		return nil, &Error{nil, ErrExpectElement, res}
	}

	return p.ElementFromObjectID(res.ObjectID), nil
}

// ElementsE doc is similar to the method Elements
func (p *Page) ElementsE(objectID proto.RuntimeRemoteObjectID, selector string) (Elements, error) {
	return p.ElementsByJSE(objectID, p.jsFn("elements"), Array{selector})
}

// ElementsXE doc is similar to the method ElementsX
func (p *Page) ElementsXE(objectID proto.RuntimeRemoteObjectID, xpath string) (Elements, error) {
	return p.ElementsByJSE(objectID, p.jsFn("elementsX"), Array{xpath})
}

// ElementsByJSE is different from ElementByJSE, it doesn't do retry
func (p *Page) ElementsByJSE(thisID proto.RuntimeRemoteObjectID, js string, params Array) (Elements, error) {
	res, err := p.EvalE(false, thisID, js, params)
	if err != nil {
		return nil, err
	}

	if res.Subtype != proto.RuntimeRemoteObjectSubtypeArray {
		return nil, &Error{nil, ErrExpectElements, res}
	}

	objectID := res.ObjectID
	defer func() { err = p.ReleaseE(objectID) }()

	list, err := proto.RuntimeGetProperties{
		ObjectID:      objectID,
		OwnProperties: true,
	}.Call(p)
	if err != nil {
		return nil, err
	}

	elemList := Elements{}
	for _, obj := range list.Result {
		if obj.Name == "__proto__" || obj.Name == "length" {
			continue
		}
		val := obj.Value

		if val.Subtype != proto.RuntimeRemoteObjectSubtypeNode {
			return nil, &Error{nil, ErrExpectElements, val}
		}

		elemList = append(elemList, p.ElementFromObjectID(val.ObjectID))
	}

	return elemList, err
}

// HasE doc is similar to the method Has
func (el *Element) HasE(selector string) (bool, error) {
	_, err := el.ElementE(selector)
	if IsError(err, ErrElementNotFound) {
		return false, nil
	}
	return err == nil, err
}

// HasXE doc is similar to the method HasX
func (el *Element) HasXE(selector string) (bool, error) {
	_, err := el.ElementXE(selector)
	if IsError(err, ErrElementNotFound) {
		return false, nil
	}
	return err == nil, err
}

// HasMatchesE doc is similar to the method HasMatches
func (el *Element) HasMatchesE(selector, regex string) (bool, error) {
	_, err := el.ElementMatchesE(selector, regex)
	if IsError(err, ErrElementNotFound) {
		return false, nil
	}
	return err == nil, err
}

// ElementE doc is similar to the method Element
func (el *Element) ElementE(selector string) (*Element, error) {
	return el.page.ElementE(nil, el.ObjectID, selector)
}

// ElementXE doc is similar to the method ElementX
func (el *Element) ElementXE(xpath string) (*Element, error) {
	return el.page.ElementXE(nil, el.ObjectID, xpath)
}

// ElementByJSE doc is similar to the method ElementByJS
func (el *Element) ElementByJSE(js string, params Array) (*Element, error) {
	return el.page.ElementByJSE(nil, el.ObjectID, js, params)
}

// ParentE doc is similar to the method Parent
func (el *Element) ParentE() (*Element, error) {
	return el.ElementByJSE(`() => this.parentElement`, nil)
}

// ParentsE that match the selector
func (el *Element) ParentsE(selector string) (Elements, error) {
	return el.ElementsByJSE(el.page.jsFn("parents"), Array{selector})
}

// NextE doc is similar to the method Next
func (el *Element) NextE() (*Element, error) {
	return el.ElementByJSE(`() => this.nextElementSibling`, nil)
}

// PreviousE doc is similar to the method Previous
func (el *Element) PreviousE() (*Element, error) {
	return el.ElementByJSE(`() => this.previousElementSibling`, nil)
}

// ElementMatchesE doc is similar to the method ElementMatches
func (el *Element) ElementMatchesE(selector, regex string) (*Element, error) {
	return el.page.ElementMatchesE(nil, el.ObjectID, selector, regex)
}

// ElementsE doc is similar to the method Elements
func (el *Element) ElementsE(selector string) (Elements, error) {
	return el.page.ElementsE(el.ObjectID, selector)
}

// ElementsXE doc is similar to the method ElementsX
func (el *Element) ElementsXE(xpath string) (Elements, error) {
	return el.page.ElementsXE(el.ObjectID, xpath)
}

// ElementsByJSE doc is similar to the method ElementsByJS
func (el *Element) ElementsByJSE(js string, params Array) (Elements, error) {
	return el.page.ElementsByJSE(el.ObjectID, js, params)
}
