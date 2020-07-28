// This file contains all query related code for Page and Element to separate the concerns.

package rod

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/go-rod/rod/lib/proto"
	"github.com/ysmood/kit"
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
		res, err := page.EvalE(true, "", `location.href`, nil)
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
func (p *Page) HasE(selectors ...string) (bool, error) {
	_, err := p.ElementE(nil, "", selectors)
	if errors.Is(err, ErrElementNotFound) {
		return false, nil
	}
	return err == nil, err
}

// HasXE doc is similar to the method HasX
func (p *Page) HasXE(selectors ...string) (bool, error) {
	_, err := p.ElementXE(nil, "", selectors)
	if errors.Is(err, ErrElementNotFound) {
		return false, nil
	}
	return err == nil, err
}

// HasMatchesE doc is similar to the method HasMatches
func (p *Page) HasMatchesE(pairs ...string) (bool, error) {
	_, err := p.ElementMatchesE(nil, "", pairs)
	if errors.Is(err, ErrElementNotFound) {
		return false, nil
	}
	return err == nil, err
}

// ElementE finds element by css selector
func (p *Page) ElementE(sleeper kit.Sleeper, objectID proto.RuntimeRemoteObjectID, selectors []string) (*Element, error) {
	js, jsArgs := jsHelper("element", ArrayFromList(selectors))
	return p.ElementByJSE(sleeper, objectID, js, jsArgs)
}

// ElementMatchesE doc is similar to the method ElementMatches
func (p *Page) ElementMatchesE(sleeper kit.Sleeper, objectID proto.RuntimeRemoteObjectID, pairs []string) (*Element, error) {
	js, jsArgs := jsHelper("elementMatches", ArrayFromList(pairs))
	return p.ElementByJSE(sleeper, objectID, js, jsArgs)
}

// ElementXE finds elements by XPath
func (p *Page) ElementXE(sleeper kit.Sleeper, objectID proto.RuntimeRemoteObjectID, xPaths []string) (*Element, error) {
	js, jsArgs := jsHelper("elementX", ArrayFromList(xPaths))
	return p.ElementByJSE(sleeper, objectID, js, jsArgs)
}

// ElementByJSE returns the element from the return value of the js function.
// sleeper is used to sleep before retry the operation.
// If sleeper is nil, no retry will be performed.
// thisID is the this value of the js function, when thisID is "", the this context will be the "window".
// If the js function returns "null", ElementByJSE will retry, you can use custom sleeper to make it only
// retry once.
func (p *Page) ElementByJSE(sleeper kit.Sleeper, thisID proto.RuntimeRemoteObjectID, js string, params Array) (*Element, error) {
	var res *proto.RuntimeRemoteObject
	var err error

	if sleeper == nil {
		sleeper = func(_ context.Context) error {
			return fmt.Errorf("%w by js: %s", newErr(ErrElementNotFound, js), js)
		}
	}

	removeTrace := func() {}
	err = kit.Retry(p.ctx, sleeper, func() (bool, error) {
		remove := p.tryTraceFn(js, params)
		removeTrace()
		removeTrace = remove

		res, err = p.EvalE(false, thisID, js, params)
		if err != nil {
			return true, err
		}

		if res.Type == proto.RuntimeRemoteObjectTypeObject && res.Subtype == proto.RuntimeRemoteObjectSubtypeNull {
			return false, nil
		}

		return true, nil
	})
	removeTrace()
	if err != nil {
		return nil, err
	}

	if res.Subtype != proto.RuntimeRemoteObjectSubtypeNode {
		return nil, fmt.Errorf("%w but got: %s", newErr(ErrExpectElement, res), kit.MustToJSON(res))
	}

	return p.ElementFromObject(res.ObjectID), nil
}

// ElementsE doc is similar to the method Elements
func (p *Page) ElementsE(objectID proto.RuntimeRemoteObjectID, selector string) (Elements, error) {
	js, jsArgs := jsHelper("elements", Array{selector})
	return p.ElementsByJSE(objectID, js, jsArgs)
}

// ElementsXE doc is similar to the method ElementsX
func (p *Page) ElementsXE(objectID proto.RuntimeRemoteObjectID, xpath string) (Elements, error) {
	js, jsArgs := jsHelper("elementsX", Array{xpath})
	return p.ElementsByJSE(objectID, js, jsArgs)
}

// ElementsByJSE is different from ElementByJSE, it doesn't do retry
func (p *Page) ElementsByJSE(thisID proto.RuntimeRemoteObjectID, js string, params Array) (Elements, error) {
	res, err := p.EvalE(false, thisID, js, params)
	if err != nil {
		return nil, err
	}

	if res.Subtype != proto.RuntimeRemoteObjectSubtypeArray {
		return nil, fmt.Errorf("%w but got: %s", newErr(ErrExpectElements, res), kit.MustToJSON(res))
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
			return nil, fmt.Errorf("%w: %s", newErr(ErrExpectElements, val), kit.MustToJSON(val))
		}

		elemList = append(elemList, p.ElementFromObject(val.ObjectID))
	}

	return elemList, err
}

// SearchE for each given query in the DOM tree until the result count is not zero, before that it will keep retrying.
// The query can be plain text or css selector or xpath.
// It will search nested iframes and shadow doms too.
func (p *Page) SearchE(sleeper kit.Sleeper, queries []string, from, to int) (Elements, error) {
	if sleeper == nil {
		sleeper = func(_ context.Context) error {
			return fmt.Errorf("%w by query: %v", newErr(ErrElementNotFound, queries), queries)
		}
	}

	list := Elements{}

	search := func(query string) (bool, error) {
		search, err := proto.DOMPerformSearch{
			Query:                     query,
			IncludeUserAgentShadowDOM: true,
		}.Call(p)
		defer func() {
			_ = proto.DOMDiscardSearchResults{SearchID: search.SearchID}.Call(p)
		}()
		if err != nil {
			return true, err
		}

		if search.ResultCount == 0 {
			return false, nil
		}

		result, err := proto.DOMGetSearchResults{
			SearchID:  search.SearchID,
			FromIndex: int64(from),
			ToIndex:   int64(to),
		}.Call(p)
		if err != nil {
			if isNilContextErr(err) {
				return false, nil
			}
			return true, err
		}

		for _, id := range result.NodeIds {
			if id == 0 {
				return false, nil
			}

			el, err := p.ElementFromNodeE(id)
			if err != nil {
				return true, err
			}
			list = append(list, el)
		}

		return true, nil
	}

	err := kit.Retry(p.ctx, sleeper, func() (bool, error) {
		p.enableNodeQuery()

		for _, query := range queries {
			stop, err := search(query)
			if stop {
				return stop, err
			}
		}
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	return list, nil
}

// HasE doc is similar to the method Has
func (el *Element) HasE(selector string) (bool, error) {
	_, err := el.ElementE(selector)
	if errors.Is(err, ErrElementNotFound) {
		return false, nil
	}
	return err == nil, err
}

// HasXE doc is similar to the method HasX
func (el *Element) HasXE(selector string) (bool, error) {
	_, err := el.ElementXE(selector)
	if errors.Is(err, ErrElementNotFound) {
		return false, nil
	}
	return err == nil, err
}

// HasMatchesE doc is similar to the method HasMatches
func (el *Element) HasMatchesE(selector, regex string) (bool, error) {
	_, err := el.ElementMatchesE(selector, regex)
	if errors.Is(err, ErrElementNotFound) {
		return false, nil
	}
	return err == nil, err
}

// ElementE doc is similar to the method Element
func (el *Element) ElementE(selectors ...string) (*Element, error) {
	return el.page.ElementE(nil, el.ObjectID, selectors)
}

// ElementXE doc is similar to the method ElementX
func (el *Element) ElementXE(xPaths ...string) (*Element, error) {
	return el.page.ElementXE(nil, el.ObjectID, xPaths)
}

// ElementByJSE doc is similar to the method ElementByJS
func (el *Element) ElementByJSE(js string, params Array) (*Element, error) {
	return el.page.ElementByJSE(nil, el.ObjectID, js, params)
}

// ParentE doc is similar to the method Parent
func (el *Element) ParentE() (*Element, error) {
	return el.ElementByJSE(`this.parentElement`, nil)
}

// ParentsE that match the selector
func (el *Element) ParentsE(selector string) (Elements, error) {
	js, params := jsHelper("parents", Array{selector})
	return el.ElementsByJSE(js, params)
}

// NextE doc is similar to the method Next
func (el *Element) NextE() (*Element, error) {
	return el.ElementByJSE(`this.nextElementSibling`, nil)
}

// PreviousE doc is similar to the method Previous
func (el *Element) PreviousE() (*Element, error) {
	return el.ElementByJSE(`this.previousElementSibling`, nil)
}

// ElementMatchesE doc is similar to the method ElementMatches
func (el *Element) ElementMatchesE(pairs ...string) (*Element, error) {
	return el.page.ElementMatchesE(nil, el.ObjectID, pairs)
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
