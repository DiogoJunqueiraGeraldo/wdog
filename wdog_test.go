package wdog_test

import (
	"context"
	"github.com/DiogoJunqueiraGeraldo/wdog"
	"testing"
	"time"
)

type OwnerFake struct {
	NoiseMemory []wdog.Noise
}

func NewOwnerFake() *OwnerFake {
	return &OwnerFake{
		NoiseMemory: make([]wdog.Noise, 0, 1024),
	}
}

func (o *OwnerFake) Hear(noise wdog.Noise) {
	o.NoiseMemory = append(o.NoiseMemory, noise)
}

func NotCtxCompliant(_ context.Context) {
	time.Sleep(time.Second * 1)
}

func Panic(_ context.Context) {
	panic("panic")
}

func CtxCompliant(ctx context.Context) {
	select {
	case <-ctx.Done():
	case <-time.After(time.Second * 1):
	}
}

func Test_NotCtxCompliant_WithoutCancel(t *testing.T) {
	ow := NewOwnerFake()
	wd := wdog.New(wdog.NewConfiguration(ow))
	wd.Watch()

	ctx := context.Background()
	wd.Go(ctx, NotCtxCompliant)

	// Teardown time
	time.Sleep(time.Millisecond * 500)

	if len(ow.NoiseMemory) != 0 {
		t.Fatalf("Unexpected noise memory size, want %d, got %d", 0, len(ow.NoiseMemory))
	}
}

func Test_NotCtxCompliant_WithCancel(t *testing.T) {
	ow := NewOwnerFake()
	wd := wdog.New(wdog.NewConfiguration(ow))
	wd.Watch()

	ctx, cancel := context.WithCancel(context.Background())
	wd.Go(ctx, NotCtxCompliant)
	cancel()

	// Teardown time
	time.Sleep(time.Millisecond * 500)

	if len(ow.NoiseMemory) != 1 {
		t.Fatalf("Unexpected noise memory size, want %d, got %d", 1, len(ow.NoiseMemory))
	}
}

func Test_NotCtxCompliant_WithCancel_MultipleTimes(t *testing.T) {
	ow := NewOwnerFake()
	wd := wdog.New(wdog.NewConfiguration(ow))
	wd.Watch()

	ctx, cancel := context.WithCancel(context.Background())
	wd.Go(ctx, NotCtxCompliant)
	wd.Go(ctx, NotCtxCompliant)
	wd.Go(ctx, NotCtxCompliant)
	cancel()

	// Teardown time
	time.Sleep(time.Millisecond * 500)

	if len(ow.NoiseMemory) != 4 {
		t.Fatalf("Unexpected noise memory size, want %d, got %d", 1, len(ow.NoiseMemory))
	}

	if ow.NoiseMemory[3].Type != wdog.Bark {
		t.Fatalf("Unexpected noise type, want %s, got %s", wdog.Bark, ow.NoiseMemory[3].Type)
	}
}

func Test_CtxCompliant_WithCancel_MultipleTimes(t *testing.T) {
	ow := NewOwnerFake()
	wd := wdog.New(wdog.NewConfiguration(ow))
	wd.Watch()

	ctx, cancel := context.WithCancel(context.Background())
	wd.Go(ctx, CtxCompliant)
	wd.Go(ctx, CtxCompliant)
	wd.Go(ctx, CtxCompliant)
	cancel()

	// Teardown time
	time.Sleep(time.Millisecond * 500)

	if len(ow.NoiseMemory) != 0 {
		t.Fatalf("Unexpected noise memory size, want %d, got %d", 1, len(ow.NoiseMemory))
	}
}

func Test_Panic_MultipleTimes(t *testing.T) {
	ow := NewOwnerFake()
	wd := wdog.New(wdog.NewConfiguration(ow))
	wd.Watch()

	ctx, cancel := context.WithCancel(context.Background())
	wd.Go(ctx, Panic)
	wd.Go(ctx, Panic)
	wd.Go(ctx, Panic)
	cancel()

	// Teardown time
	time.Sleep(time.Millisecond * 500)

	if len(ow.NoiseMemory) != 3 {
		t.Fatalf("Unexpected noise memory size, want %d, got %d", 3, len(ow.NoiseMemory))
	}
}
