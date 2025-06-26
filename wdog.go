package wdog

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrToleranceExceeded       = errors.New("watchdog tolerance exceeded")
	ErrTaskPanicked            = errors.New("task under watch panicked")
	ErrTaskNotContextCompliant = errors.New("task not context compliant")
)

type NoiseType string

const (
	Growl NoiseType = "growl"
	Bark  NoiseType = "bark"
	Cry   NoiseType = "cry"
)

type WDog struct {
	m sync.RWMutex

	owner Owner

	errCount    int32
	hall        chan Noise
	hallTimeout time.Duration
	ticker      *time.Ticker

	teardownTimeout time.Duration

	toleranceWindow time.Duration
	toleranceCap    int32
}

type Noise struct {
	Type NoiseType

	ErrCount int32
	Error    error

	Payload any
}

type Owner interface {
	Hear(context.Context, Noise)
}

func New(ow Owner) *WDog {
	hall := make(chan Noise, 1024)

	return &WDog{
		errCount:        0,
		hall:            hall,
		hallTimeout:     time.Millisecond * 5,
		owner:           ow,
		teardownTimeout: time.Millisecond * 50,
		toleranceWindow: time.Millisecond * 100,
		toleranceCap:    2,
	}
}

func (w *WDog) WithTeardownTimeout(d time.Duration) {
	if w.ticker != nil {
		panic("watchdog already working")
	}

	w.m.Lock()
	defer w.m.Unlock()
	w.teardownTimeout = d
}

func (w *WDog) WithToleranceWindow(d time.Duration) {
	if w.ticker != nil {
		panic("watchdog already working")
	}

	w.m.Lock()
	defer w.m.Unlock()
	w.toleranceWindow = d
}

func (w *WDog) WithToleranceCap(d int32) {
	if w.ticker != nil {
		panic("watchdog already working")
	}

	w.m.Lock()
	defer w.m.Unlock()
	w.toleranceCap = d
}

func (w *WDog) Watch(ctx context.Context) {
	go w.monitorTolerance(ctx)
	go w.listenToHall(ctx)
}

func (w *WDog) monitorTolerance(ctx context.Context) {
	w.m.Lock()
	w.ticker = time.NewTicker(w.toleranceWindow)
	defer w.ticker.Stop()
	w.m.Unlock()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.ticker.C:
			errCountSnapshot := atomic.LoadInt32(&w.errCount)
			if errCountSnapshot >= w.toleranceCap {
				w.emitNoise(ctx, Noise{
					Type:     Bark,
					ErrCount: errCountSnapshot,
					Error:    ErrToleranceExceeded,
				})

				atomic.StoreInt32(&w.errCount, 0)
			}
		}
	}
}

func (w *WDog) emitNoise(ctx context.Context, noise Noise) {
	select {
	case w.hall <- noise:
	case <-time.After(w.teardownTimeout):
	case <-ctx.Done():
	}
}

func (w *WDog) listenToHall(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case noise := <-w.hall:
			go w.owner.Hear(ctx, noise)
		}
	}
}

func (w *WDog) Go(ctx context.Context, t func(ctx context.Context)) {
	done := make(chan struct{})

	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				atomic.AddInt32(&w.errCount, 1)
				w.emitNoise(ctx, Noise{
					Type:    Cry,
					Error:   ErrTaskPanicked,
					Payload: r,
				})
			}
		}()
		t(ctx)
	}()

	select {
	case <-done:
		return
	case <-ctx.Done():
		select {
		case <-done:
			return
		case <-time.After(w.teardownTimeout):
			atomic.AddInt32(&w.errCount, 1)
			go w.emitNoise(ctx, Noise{
				Type:  Growl,
				Error: ErrTaskNotContextCompliant,
			})
			return
		}
	}
}
