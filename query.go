// This file contains all query related code for Page and Element to separate the concerns.

package rod

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/go-rod/rod/lib/assets/js"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
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

// First returns the first page, if the list is empty returns nil
func (ps Pages) First() *Page {
	if ps.Empty() {
		return nil
	}
	return ps[0]
}

// Last returns the last page, if the list is empty returns nil
func (ps Pages) Last() *Page {
	if ps.Empty() {
		return nil
	}
	return ps[len(ps)-1]
}

// Empty returns true if the list is empty
func (ps Pages) Empty() bool {
	return len(ps) == 0
}

// Find the page that has the specified element with the css selector
func (ps Pages) Find(selector string) (*Page, error) {
	for _, page := range ps {
		has, _, err := page.Has(selector)
		if err != nil {
			return nil, err
		}
		if has {
			return page, nil
		}
	}
	return nil, nil
}

// FindByURL returns the page that has the url that matches the js regex
func (ps Pages) FindByURL(regex string) (*Page, error) {
	for _, page := range ps {
		res, err := page.Eval(`location.href`)
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

// Has an element that matches the css selector
func (p *Page) Has(selectors ...string) (bool, *Element, error) {
	el, err := p.Sleeper(nil).Element(selectors...)
	if errors.Is(err, ErrElementNotFound) {
		return false, nil, nil
	}
	return err == nil, el, err
}

// HasX an element that matches the XPath selector
func (p *Page) HasX(selectors ...string) (bool, *Element, error) {
	el, err := p.Sleeper(nil).ElementX(selectors...)
	if errors.Is(err, ErrElementNotFound) {
		return false, nil, nil
	}
	return err == nil, el, err
}

// HasR an element that matches the css selector and its display text matches the js regex.
func (p *Page) HasR(selector, regex string) (bool, *Element, error) {
	el, err := p.Sleeper(nil).ElementR(selector, regex)
	if errors.Is(err, ErrElementNotFound) {
		return false, nil, nil
	}
	return err == nil, el, err
}

// Element retries until an element in the page that matches one of the CSS selectors, then returns
// the matched element.
func (p *Page) Element(selectors ...string) (*Element, error) {
	return p.ElementByJS(jsHelper(js.Element, JSArgsFromString(selectors)))
}

// ElementR retries until an element in the page that matches one of the pairs, then returns
// the matched element.
// Each pairs is a css selector and a regex. A sample call will look like page.MustElementR("div", "click me").
// The regex is the js regex, not golang's.
func (p *Page) ElementR(pairs ...string) (*Element, error) {
	return p.ElementByJS(jsHelper(js.ElementR, JSArgsFromString(pairs)))
}

// ElementX retries until an element in the page that matches one of the XPath selectors, then returns
// the matched element.
func (p *Page) ElementX(xPaths ...string) (*Element, error) {
	return p.ElementByJS(jsHelper(js.ElementX, JSArgsFromString(xPaths)))
}

// ElementByJS returns the element from the return value of the js function.
// If sleeper is nil, no retry will be performed.
// thisID is the this value of the js function, when thisID is "", the this context will be the "window".
// If the js function returns "null", ElementByJS will retry, you can use custom sleeper to make it only
// retry once.
func (p *Page) ElementByJS(opts *EvalOptions) (*Element, error) {
	var res *proto.RuntimeRemoteObject
	var err error

	sleeper := p.sleeper()
	if sleeper == nil {
		sleeper = func(context.Context) error {
			return newErr(ErrElementNotFound, opts, opts.JS)
		}
	}

	removeTrace := func() {}
	err = utils.Retry(p.ctx, sleeper, func() (bool, error) {
		remove := p.tryTraceEval(opts.JS, opts.JSArgs)
		removeTrace()
		removeTrace = remove

		res, err = p.EvalWithOptions(opts.ByObject())
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
		return nil, newErr(ErrExpectElement, res, utils.MustToJSON(res))
	}

	return p.ElementFromObject(res.ObjectID), nil
}

// Elements returns all elements that match the css selector
func (p *Page) Elements(selector string) (Elements, error) {
	return p.ElementsByJS(jsHelper(js.Elements, JSArgs{selector}))
}

// ElementsX returns all elements that match the XPath selector
func (p *Page) ElementsX(xpath string) (Elements, error) {
	return p.ElementsByJS(jsHelper(js.ElementsX, JSArgs{xpath}))
}

// ElementsByJS returns the elements from the return value of the js
func (p *Page) ElementsByJS(opts *EvalOptions) (Elements, error) {
	res, err := p.EvalWithOptions(opts.ByObject())
	if err != nil {
		return nil, err
	}

	if res.Subtype != proto.RuntimeRemoteObjectSubtypeArray {
		return nil, newErr(ErrExpectElements, res, utils.MustToJSON(res))
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
			return nil, newErr(ErrExpectElements, val, utils.MustToJSON(val))
		}

		elemList = append(elemList, p.ElementFromObject(val.ObjectID))
	}

	return elemList, err
}

// Search for each given query in the DOM tree until the result count is not zero, before that it will keep retrying.
// The query can be plain text or css selector or xpath.
// It will search nested iframes and shadow doms too.
func (p *Page) Search(from, to int, queries ...string) (Elements, error) {
	sleeper := p.sleeper()
	if sleeper == nil {
		sleeper = func(context.Context) error {
			return newErr(ErrElementNotFound, queries, fmt.Sprintf("%v", queries))
		}
	}

	list := Elements{}

	search := func(query string) (bool, error) {
		search, err := proto.DOMPerformSearch{
			Query:                     query,
			IncludeUserAgentShadowDOM: true,
		}.Call(p)
		if err != nil {
			return true, err
		}

		defer func() {
			_ = proto.DOMDiscardSearchResults{SearchID: search.SearchID}.Call(p)
		}()

		if search.ResultCount == 0 {
			return false, nil
		}

		result, err := proto.DOMGetSearchResults{
			SearchID:  search.SearchID,
			FromIndex: int64(from),
			ToIndex:   int64(to),
		}.Call(p)
		if err != nil {
			// when the page is still loading the search result is not ready
			if isNilContextErr(err) {
				return false, nil
			}
			return true, err
		}

		for _, id := range result.NodeIds {
			// TODO: some times the node id can be zero, feels like a bug of devtools server
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

	err := utils.Retry(p.ctx, sleeper, func() (bool, error) {
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

type raceBranch struct {
	condition func() (*Element, error)
	callback  func(*Element) error
}

// RaceContext stores the branches to race
type RaceContext struct {
	page        *Page
	noSleepPage *Page
	branches    []*raceBranch
}

// Race creates a context to race selectors
func (p *Page) Race() *RaceContext {
	return &RaceContext{page: p, noSleepPage: p.Sleeper(nil)}
}

// Element the doc is similar with MustElement but has a callback when a match is found
func (rc *RaceContext) Element(selector string, callback func(*Element) error) *RaceContext {
	rc.branches = append(rc.branches, &raceBranch{
		func() (*Element, error) { return rc.noSleepPage.Element(selector) },
		callback,
	})
	return rc
}

// ElementX the doc is similar with ElementX but has a callback when a match is found
func (rc *RaceContext) ElementX(selector string, callback func(*Element) error) *RaceContext {
	rc.branches = append(rc.branches, &raceBranch{
		func() (*Element, error) { return rc.noSleepPage.ElementX(selector) },
		callback,
	})
	return rc
}

// ElementR the doc is similar with ElementR but has a callback when a match is found
func (rc *RaceContext) ElementR(selector, regex string, callback func(*Element) error) *RaceContext {
	rc.branches = append(rc.branches, &raceBranch{
		func() (*Element, error) { return rc.noSleepPage.ElementR(selector, regex) },
		callback,
	})
	return rc
}

// ElementByJS the doc is similar with MustElementByJS but has a callback when a match is found
func (rc *RaceContext) ElementByJS(opts *EvalOptions, callback func(*Element) error) *RaceContext {
	rc.branches = append(rc.branches, &raceBranch{
		func() (*Element, error) { return rc.noSleepPage.ElementByJS(opts) },
		callback,
	})
	return rc
}

// Do the race
func (rc *RaceContext) Do() error {
	return utils.Retry(rc.page.ctx, rc.page.sleeper(), func() (stop bool, err error) {
		for _, branch := range rc.branches {
			el, err := branch.condition()
			if err == nil {
				return true, branch.callback(el)
			} else if !errors.Is(err, ErrElementNotFound) {
				return true, err
			}
		}
		return
	})
}

// Has an element that matches the css selector
func (el *Element) Has(selector string) (bool, *Element, error) {
	el, err := el.Element(selector)
	if errors.Is(err, ErrElementNotFound) {
		return false, nil, nil
	}
	return err == nil, el, err
}

// HasX an element that matches the XPath selector
func (el *Element) HasX(selector string) (bool, *Element, error) {
	el, err := el.ElementX(selector)
	if errors.Is(err, ErrElementNotFound) {
		return false, nil, nil
	}
	return err == nil, el, err
}

// HasR an element that matches the css selector and its text matches the js regex.
func (el *Element) HasR(selector, regex string) (bool, *Element, error) {
	el, err := el.ElementR(selector, regex)
	if errors.Is(err, ErrElementNotFound) {
		return false, nil, nil
	}
	return err == nil, el, err
}

// Element returns the first child that matches the css selector
func (el *Element) Element(selectors ...string) (*Element, error) {
	return el.ElementByJS(jsHelper(js.Element, JSArgsFromString(selectors)))
}

// ElementX returns the first child that matches the XPath selector
func (el *Element) ElementX(xPaths ...string) (*Element, error) {
	return el.ElementByJS(jsHelper(js.ElementX, JSArgsFromString(xPaths)))
}

// ElementByJS returns the element from the return value of the js
func (el *Element) ElementByJS(opts *EvalOptions) (*Element, error) {
	return el.page.Sleeper(nil).ElementByJS(opts.This(el.ObjectID))
}

// Parent returns the parent element in the DOM tree
func (el *Element) Parent() (*Element, error) {
	return el.ElementByJS(NewEvalOptions(`this.parentElement`, nil))
}

// Parents that match the selector
func (el *Element) Parents(selector string) (Elements, error) {
	return el.ElementsByJS(jsHelper(js.Parents, JSArgs{selector}))
}

// Next returns the next sibling element in the DOM tree
func (el *Element) Next() (*Element, error) {
	return el.ElementByJS(NewEvalOptions(`this.nextElementSibling`, nil))
}

// Previous returns the previous sibling element in the DOM tree
func (el *Element) Previous() (*Element, error) {
	return el.ElementByJS(NewEvalOptions(`this.previousElementSibling`, nil))
}

// ElementR returns the first element in the page that matches the CSS selector and its text matches the js regex.
func (el *Element) ElementR(pairs ...string) (*Element, error) {
	return el.ElementByJS(jsHelper(js.ElementR, JSArgsFromString(pairs)))
}

// Elements returns all elements that match the css selector
func (el *Element) Elements(selector string) (Elements, error) {
	return el.ElementsByJS(jsHelper(js.Elements, JSArgs{selector}))
}

// ElementsX returns all elements that match the XPath selector
func (el *Element) ElementsX(xpath string) (Elements, error) {
	return el.ElementsByJS(jsHelper(js.ElementsX, JSArgs{xpath}))
}

// ElementsByJS returns the elements from the return value of the js
func (el *Element) ElementsByJS(opts *EvalOptions) (Elements, error) {
	return el.page.Context(el.ctx).Sleeper(nil).ElementsByJS(opts.This(el.ObjectID))
}
