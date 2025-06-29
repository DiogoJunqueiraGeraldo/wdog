package wdog_test

import (
	"context"
	"errors"
	"fmt"
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

func (o *OwnerFake) LastNoise() wdog.Noise {
	noiseLen := len(o.NoiseMemory)
	if noiseLen == 0 {
		return wdog.Silence
	}

	return o.NoiseMemory[noiseLen-1]
}

func (o *OwnerFake) DiffHistory(expect []wdog.NoiseType) error {
	if len(o.NoiseMemory) < len(expect) {
		msg := fmt.Sprintf("Missing expected noises, expected len %d, got %d", len(expect), len(o.NoiseMemory))
		return errors.New(msg)
	}

	if len(o.NoiseMemory) > len(expect) {
		msg := fmt.Sprintf("More than expected noises, expected len %d, got %d", len(expect), len(o.NoiseMemory))
		return errors.New(msg)
	}

	var err error
	for i, noise := range o.NoiseMemory {
		if noise.Type != expect[i] {
			msg := fmt.Sprintf("Mismatch noise type at %d, want %s, got %s", i, expect[i], noise.Type)
			err = errors.Join(err, errors.New(msg))
		}
	}

	return err
}

func nonCompliantTask(_ context.Context) {
	time.Sleep(time.Second * 1)
}

func panicTask(_ context.Context) {
	panic("panic")
}

func compliantTask(ctx context.Context) {
	select {
	case <-ctx.Done():
	case <-time.After(time.Second * 1):
	}
}

func TestNonCompliantTaskWithoutCancel(t *testing.T) {
	t.Parallel()

	ow := NewOwnerFake()
	wd := wdog.New(wdog.NewConfiguration(ow))
	wd.Watch()
	defer wd.Close()

	ctx := context.Background()
	wd.Go(ctx, nonCompliantTask)

	// Teardown time
	time.Sleep(time.Millisecond * 500)

	if ow.LastNoise() != wdog.Silence {
		t.Error("Unexpected noise value")
	}
}

func TestNonCompliantTaskWithCancel(t *testing.T) {
	t.Parallel()

	ow := NewOwnerFake()
	wd := wdog.New(wdog.NewConfiguration(ow))
	wd.Watch()
	defer wd.Close()

	ctx, cancel := context.WithCancel(context.Background())
	wd.Go(ctx, nonCompliantTask)
	cancel()

	// Teardown time
	time.Sleep(time.Millisecond * 500)

	err := ow.DiffHistory([]wdog.NoiseType{wdog.Growl})
	if err != nil {
		t.Error(err)
	}
}

func TestNonCompliantTaskWithCancelMultipleTimes(t *testing.T) {
	t.Parallel()

	ow := NewOwnerFake()
	wd := wdog.New(wdog.NewConfiguration(ow))
	wd.Watch()
	defer wd.Close()

	ctx, cancel := context.WithCancel(context.Background())
	wd.Go(ctx, nonCompliantTask)
	wd.Go(ctx, nonCompliantTask)
	wd.Go(ctx, nonCompliantTask)
	cancel()

	// Teardown time
	time.Sleep(time.Millisecond * 500)

	err := ow.DiffHistory([]wdog.NoiseType{wdog.Growl, wdog.Growl, wdog.Growl, wdog.Bark})
	if err != nil {
		t.Error(err)
	}
}

func TestCompliantTaskWithCancelMultipleTimes(t *testing.T) {
	t.Parallel()

	ow := NewOwnerFake()
	wd := wdog.New(wdog.NewConfiguration(ow))

	wd.Watch()
	defer wd.Close()

	ctx, cancel := context.WithCancel(context.Background())
	wd.Go(ctx, compliantTask)
	wd.Go(ctx, compliantTask)
	wd.Go(ctx, compliantTask)
	cancel()

	// Teardown time
	time.Sleep(time.Millisecond * 500)

	if ow.LastNoise() != wdog.Silence {
		t.Error("Unexpected noise value")
	}
}

func TestPanicTaskMultipleTimes(t *testing.T) {
	t.Parallel()

	ow := NewOwnerFake()
	wd := wdog.New(wdog.NewConfiguration(ow))

	wd.Watch()
	defer wd.Close()

	ctx, cancel := context.WithCancel(context.Background())
	wd.Go(ctx, panicTask)
	wd.Go(ctx, panicTask)
	wd.Go(ctx, panicTask)
	cancel()

	// Teardown time
	time.Sleep(time.Millisecond * 500)

	if len(ow.NoiseMemory) != 3 {
		t.Fatalf("Unexpected noise memory size, want %d, got %d", 3, len(ow.NoiseMemory))
	}

	err := ow.DiffHistory([]wdog.NoiseType{wdog.Cry, wdog.Cry, wdog.Cry})
	if err != nil {
		t.Error(err)
	}
}
