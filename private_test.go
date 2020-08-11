package rod

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/go-rod/rod/lib/defaults"
	"github.com/stretchr/testify/suite"
)

// S test suite
type S struct {
	suite.Suite
}

func Test(t *testing.T) {
	s := new(S)
	suite.Run(t, s)
}

func (s *S) TestDefaultTraceLoggers() {
	defaultTraceLogAct("msg")
	defaultTraceLogJS("fn", Array{1, 2})
	defaultTraceLogErr(errors.New("err"))
}

func (s *S) TestUpdateMouseTracerErr() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cdpCall := func(ctx context.Context, sessionID, method string, params interface{}) ([]byte, error) {
		return nil, errors.New("err")
	}
	m := &Mouse{page: &Page{ctx: ctx, lock: &sync.Mutex{}, browser: &Browser{cdpCall: cdpCall}}}

	s.True(m.updateMouseTracer())
}

func (s *S) TestBrowserErrs() {
	b := New()

	oldRemote := defaults.Remote
	defaults.Remote = true
	defer func() {
		defaults.Remote = oldRemote
	}()
	s.Error(b.Connect())

	b = &Browser{cdpCall: func(ctx context.Context, sessionID, method string, params interface{}) ([]byte, error) {
		return nil, errors.New("err")
	}}

	_, err := b.pageInfo("")
	s.Error(err)
}

func (s *S) TestMatchWithFilter() {
	s.False(matchWithFilter("", nil, nil))
}
