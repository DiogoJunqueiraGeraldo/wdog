package wdog

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
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

	debug bool
}

type Noise struct {
	Type NoiseType

	ErrCount int32
	Error    error

	Payload any
}

type Owner interface {
	Hear(Noise)
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
		debug:           os.Getenv("WDOG_DEBUG") == "true",
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

func (w *WDog) Watch() {
	go w.monitorTolerance()
	go w.listenToHall()
}

func (w *WDog) monitorTolerance() {
	w.m.Lock()
	w.ticker = time.NewTicker(w.toleranceWindow)
	defer w.ticker.Stop()
	w.m.Unlock()

	for {
		select {
		case <-w.ticker.C:
			errCountSnapshot := atomic.LoadInt32(&w.errCount)
			w.log(fmt.Sprintf("monitoring tolerance err count: %d", errCountSnapshot))

			if errCountSnapshot >= w.toleranceCap {
				w.log("tolerance exceeded")
				go w.emitNoise(Noise{
					Type:     Bark,
					ErrCount: errCountSnapshot,
					Error:    ErrToleranceExceeded,
				})

				atomic.StoreInt32(&w.errCount, 0)
			}
		}
	}
}

func (w *WDog) emitNoise(noise Noise) {
	select {
	case w.hall <- noise:
		w.log("emitted noise to hall")
	case <-time.After(w.teardownTimeout):
		w.log("timeout emitting noise to hall")
	}
}

func (w *WDog) listenToHall() {
	for {
		select {
		case noise := <-w.hall:
			w.log("listening to hall noise")
			go w.owner.Hear(noise)
		}
	}
}

func (w *WDog) log(msg string) {
	if w.debug {
		log.Println("DEBUG (wdog): " + msg)
	}
}

func (w *WDog) Go(ctx context.Context, t func(ctx context.Context)) {
	w.log("going up")
	done := make(chan struct{})

	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				atomic.AddInt32(&w.errCount, 1)
				go w.emitNoise(Noise{
					Type:    Cry,
					Error:   ErrTaskPanicked,
					Payload: r,
				})
				w.log("recovered from panic")
			}
		}()

		t(ctx)
		w.log("task completed")
	}()

	go func() {
		select {
		case <-done:
			w.log("task completed before ctx cancellation")
			return
		case <-ctx.Done():
			select {
			case <-done:
				w.log("task completed before teardown timeout")
				return
			case <-time.After(w.teardownTimeout):
				w.log("task teardown timeout")
				atomic.AddInt32(&w.errCount, 1)
				go w.emitNoise(Noise{
					Type:  Growl,
					Error: ErrTaskNotContextCompliant,
				})
				return
			}
		}
	}()
}
