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
		if page.MustHas(selector) {
			return page
		}
	}
	return nil
}

// FindByURL returns the page that has the url that matches the regex
func (ps Pages) FindByURL(regex string) (*Page, error) {
	for _, page := range ps {
		res, err := page.Eval(true, "", `location.href`, nil)
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

// Has doc is similar to the method MustHas
func (p *Page) Has(selectors ...string) (bool, error) {
	_, err := p.Sleeper(nil).Element("", selectors)
	if errors.Is(err, ErrElementNotFound) {
		return false, nil
	}
	return err == nil, err
}

// HasX doc is similar to the method MustHasX
func (p *Page) HasX(selectors ...string) (bool, error) {
	_, err := p.Sleeper(nil).ElementX("", selectors)
	if errors.Is(err, ErrElementNotFound) {
		return false, nil
	}
	return err == nil, err
}

// HasMatches doc is similar to the method MustHasMatches
func (p *Page) HasMatches(pairs ...string) (bool, error) {
	_, err := p.Sleeper(nil).ElementMatches("", pairs)
	if errors.Is(err, ErrElementNotFound) {
		return false, nil
	}
	return err == nil, err
}

// Element finds element by css selector
func (p *Page) Element(objectID proto.RuntimeRemoteObjectID, selectors []string) (*Element, error) {
	js, jsArgs := jsHelper("element", ArrayFromList(selectors))
	return p.ElementByJS(objectID, js, jsArgs)
}

// ElementMatches doc is similar to the method MustElementMatches
func (p *Page) ElementMatches(objectID proto.RuntimeRemoteObjectID, pairs []string) (*Element, error) {
	js, jsArgs := jsHelper("elementMatches", ArrayFromList(pairs))
	return p.ElementByJS(objectID, js, jsArgs)
}

// ElementX finds elements by XPath
func (p *Page) ElementX(objectID proto.RuntimeRemoteObjectID, xPaths []string) (*Element, error) {
	js, jsArgs := jsHelper("elementX", ArrayFromList(xPaths))
	return p.ElementByJS(objectID, js, jsArgs)
}

// ElementByJS returns the element from the return value of the js function.
// If sleeper is nil, no retry will be performed.
// thisID is the this value of the js function, when thisID is "", the this context will be the "window".
// If the js function returns "null", ElementByJS will retry, you can use custom sleeper to make it only
// retry once.
func (p *Page) ElementByJS(thisID proto.RuntimeRemoteObjectID, js string, params Array) (*Element, error) {
	var res *proto.RuntimeRemoteObject
	var err error

	sleeper := p.sleeper
	if sleeper == nil {
		sleeper = func(_ context.Context) error {
			return newErr(ErrElementNotFound, js, js)
		}
	}

	removeTrace := func() {}
	err = kit.Retry(p.ctx, sleeper, func() (bool, error) {
		remove := p.tryTraceFn(js, params)
		removeTrace()
		removeTrace = remove

		res, err = p.Eval(false, thisID, js, params)
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
		return nil, newErr(ErrExpectElement, res, kit.MustToJSON(res))
	}

	return p.ElementFromObject(res.ObjectID), nil
}

// Elements doc is similar to the method MustElements
func (p *Page) Elements(objectID proto.RuntimeRemoteObjectID, selector string) (Elements, error) {
	js, jsArgs := jsHelper("elements", Array{selector})
	return p.ElementsByJS(objectID, js, jsArgs)
}

// ElementsX doc is similar to the method MustElementsX
func (p *Page) ElementsX(objectID proto.RuntimeRemoteObjectID, xpath string) (Elements, error) {
	js, jsArgs := jsHelper("elementsX", Array{xpath})
	return p.ElementsByJS(objectID, js, jsArgs)
}

// ElementsByJS is different from ElementByJSE, it doesn't do retry
func (p *Page) ElementsByJS(thisID proto.RuntimeRemoteObjectID, js string, params Array) (Elements, error) {
	res, err := p.Eval(false, thisID, js, params)
	if err != nil {
		return nil, err
	}

	if res.Subtype != proto.RuntimeRemoteObjectSubtypeArray {
		return nil, newErr(ErrExpectElements, res, kit.MustToJSON(res))
	}

	objectID := res.ObjectID
	defer func() { err = p.Release(objectID) }()

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
			return nil, newErr(ErrExpectElements, val, kit.MustToJSON(val))
		}

		elemList = append(elemList, p.ElementFromObject(val.ObjectID))
	}

	return elemList, err
}

// Search for each given query in the DOM tree until the result count is not zero, before that it will keep retrying.
// The query can be plain text or css selector or xpath.
// It will search nested iframes and shadow doms too.
func (p *Page) Search(queries []string, from, to int) (Elements, error) {
	sleeper := p.sleeper
	if sleeper == nil {
		sleeper = func(_ context.Context) error {
			return newErr(ErrElementNotFound, queries, fmt.Sprintf("%v", queries))
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

			el, err := p.ElementFromNode(id)
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

// Has doc is similar to the method MustHas
func (el *Element) Has(selector string) (bool, error) {
	_, err := el.Element(selector)
	if errors.Is(err, ErrElementNotFound) {
		return false, nil
	}
	return err == nil, err
}

// HasX doc is similar to the method MustHasX
func (el *Element) HasX(selector string) (bool, error) {
	_, err := el.ElementX(selector)
	if errors.Is(err, ErrElementNotFound) {
		return false, nil
	}
	return err == nil, err
}

// HasMatches doc is similar to the method MustHasMatches
func (el *Element) HasMatches(selector, regex string) (bool, error) {
	_, err := el.ElementMatches(selector, regex)
	if errors.Is(err, ErrElementNotFound) {
		return false, nil
	}
	return err == nil, err
}

// Element doc is similar to the method MustElement
func (el *Element) Element(selectors ...string) (*Element, error) {
	return el.page.Sleeper(nil).Element(el.ObjectID, selectors)
}

// ElementX doc is similar to the method MustElementX
func (el *Element) ElementX(xPaths ...string) (*Element, error) {
	return el.page.Sleeper(nil).ElementX(el.ObjectID, xPaths)
}

// ElementByJS doc is similar to the method MustElementByJS
func (el *Element) ElementByJS(js string, params Array) (*Element, error) {
	return el.page.Sleeper(nil).ElementByJS(el.ObjectID, js, params)
}

// Parent doc is similar to the method MustParent
func (el *Element) Parent() (*Element, error) {
	return el.ElementByJS(`this.parentElement`, nil)
}

// Parents that match the selector
func (el *Element) Parents(selector string) (Elements, error) {
	js, params := jsHelper("parents", Array{selector})
	return el.ElementsByJS(js, params)
}

// Next doc is similar to the method MustNext
func (el *Element) Next() (*Element, error) {
	return el.ElementByJS(`this.nextElementSibling`, nil)
}

// Previous doc is similar to the method MustPrevious
func (el *Element) Previous() (*Element, error) {
	return el.ElementByJS(`this.previousElementSibling`, nil)
}

// ElementMatches doc is similar to the method MustElementMatches
func (el *Element) ElementMatches(pairs ...string) (*Element, error) {
	return el.page.Sleeper(nil).ElementMatches(el.ObjectID, pairs)
}

// Elements doc is similar to the method MustElements
func (el *Element) Elements(selector string) (Elements, error) {
	return el.page.Elements(el.ObjectID, selector)
}

// ElementsX doc is similar to the method MustElementsX
func (el *Element) ElementsX(xpath string) (Elements, error) {
	return el.page.ElementsX(el.ObjectID, xpath)
}

// ElementsByJS doc is similar to the method MustElementsByJS
func (el *Element) ElementsByJS(js string, params Array) (Elements, error) {
	return el.page.ElementsByJS(el.ObjectID, js, params)
}
