package thelp

import (
	"context"
	"time"
)

func NonCompliantTask(_ context.Context) {
	time.Sleep(time.Second * 1)
}

func PanicTask(_ context.Context) {
	panic("panic")
}

func CompliantTask(ctx context.Context) {
	select {
	case <-ctx.Done():
	case <-time.After(time.Second * 1):
	}
}
