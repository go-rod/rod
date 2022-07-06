package utils_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/go-rod/rod/lib/utils"
)

func TestBackoffSleeperWakeNow(t *testing.T) {
	g := setup(t)

	g.E(utils.BackoffSleeper(0, 0, nil)(g.Context()))
}

func TestRetry(t *testing.T) {
	g := setup(t)

	count := 0
	s1 := utils.BackoffSleeper(1, 5, nil)

	err := utils.Retry(g.Context(), s1, func() (bool, error) {
		if count > 5 {
			return true, io.EOF
		}
		count++
		return false, nil
	})

	g.Eq(err.Error(), io.EOF.Error())
}

func TestRetryCancel(t *testing.T) {
	g := setup(t)

	ctx := g.Context()
	go ctx.Cancel()
	s := utils.BackoffSleeper(time.Second, time.Second, nil)

	err := utils.Retry(ctx, s, func() (bool, error) {
		return false, nil
	})

	g.Eq(err.Error(), context.Canceled.Error())
}

func TestCountSleeperErr(t *testing.T) {
	g := setup(t)

	ctx := g.Context()
	s := utils.CountSleeper(5)
	for i := 0; i < 5; i++ {
		_ = s(ctx)
	}
	g.Err(s(ctx))
}

func TestCountSleeperCancel(t *testing.T) {
	g := setup(t)

	s := utils.CountSleeper(5)
	g.Eq(s(g.Timeout(0)), context.DeadlineExceeded)
}

func TestEachSleepers(t *testing.T) {
	g := setup(t)

	s1 := utils.BackoffSleeper(1, 5, nil)
	s2 := utils.CountSleeper(5)
	s := utils.EachSleepers(s1, s2)

	err := utils.Retry(context.Background(), s, func() (stop bool, err error) {
		return false, nil
	})

	g.Is(err, &utils.ErrMaxSleepCount{})
	g.Eq(err.Error(), "max sleep count 5 exceeded")
}

func TestRaceSleepers(t *testing.T) {
	g := setup(t)

	s1 := utils.BackoffSleeper(1, 5, nil)
	s2 := utils.CountSleeper(5)
	s := utils.RaceSleepers(s1, s2)

	err := utils.Retry(context.Background(), s, func() (stop bool, err error) {
		return false, nil
	})

	g.Is(err, &utils.ErrMaxSleepCount{})
	g.Eq(err.Error(), "max sleep count 5 exceeded")
}
