package utils_test

import (
	"context"
	"io"
	"time"

	"github.com/go-rod/rod/lib/utils"
)

func (t T) BackoffSleeperWakeNow() {
	t.E(utils.BackoffSleeper(0, 0, nil)(t.Context()))
}

func (t T) Retry() {
	count := 0
	s1 := utils.BackoffSleeper(1, 5, nil)

	err := utils.Retry(t.Context(), s1, func() (bool, error) {
		if count > 5 {
			return true, io.EOF
		}
		count++
		return false, nil
	})

	t.Eq(err.Error(), io.EOF.Error())
}

func (t T) RetryCancel() {
	ctx := t.Context()
	go ctx.Cancel()
	s := utils.BackoffSleeper(time.Second, time.Second, nil)

	err := utils.Retry(ctx, s, func() (bool, error) {
		return false, nil
	})

	t.Eq(err.Error(), context.Canceled.Error())
}

func (t T) CountSleeperErr() {
	ctx := t.Context()
	s := utils.CountSleeper(5)
	for i := 0; i < 5; i++ {
		_ = s(ctx)
	}
	t.Err(s(ctx))
}

func (t T) CountSleeperCancel() {
	s := utils.CountSleeper(5)
	t.Eq(s(t.Timeout(0)), context.DeadlineExceeded)
}

func (t T) EachSleepers() {
	s1 := utils.BackoffSleeper(1, 5, nil)
	s2 := utils.CountSleeper(5)
	s := utils.EachSleepers(s1, s2)

	err := utils.Retry(context.Background(), s, func() (stop bool, err error) {
		return false, nil
	})

	t.Is(err, &utils.ErrMaxSleepCount{})
	t.Eq(err.Error(), "max sleep count 5 exceeded")
}

func (t T) RaceSleepers() {
	s1 := utils.BackoffSleeper(1, 5, nil)
	s2 := utils.CountSleeper(5)
	s := utils.RaceSleepers(s1, s2)

	err := utils.Retry(context.Background(), s, func() (stop bool, err error) {
		return false, nil
	})

	t.Is(err, &utils.ErrMaxSleepCount{})
	t.Eq(err.Error(), "max sleep count 5 exceeded")
}
