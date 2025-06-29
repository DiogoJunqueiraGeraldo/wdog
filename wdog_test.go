package wdog_test

import (
	"context"

	"github.com/DiogoJunqueiraGeraldo/wdog"
	"github.com/DiogoJunqueiraGeraldo/wdog/internal"
	"testing"
	"time"
)

func TestNonCompliantTaskWithoutCancel(t *testing.T) {
	t.Parallel()

	ow := thelp.NewOwnerFake()
	wd := wdog.New(wdog.NewConfiguration(ow))
	wd.Watch()
	defer wd.Close()

	ctx := context.Background()
	wd.Go(ctx, thelp.NonCompliantTask)

	// Teardown time
	time.Sleep(time.Millisecond * 500)

	if ow.LastNoise() != wdog.Silence {
		t.Error("Unexpected noise value")
	}
}

func TestNonCompliantTaskWithCancel(t *testing.T) {
	t.Parallel()

	ow := thelp.NewOwnerFake()
	wd := wdog.New(wdog.NewConfiguration(ow))
	wd.Watch()
	defer wd.Close()

	ctx, cancel := context.WithCancel(context.Background())
	wd.Go(ctx, thelp.NonCompliantTask)
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

	ow := thelp.NewOwnerFake()
	wd := wdog.New(wdog.NewConfiguration(ow))
	wd.Watch()
	defer wd.Close()

	ctx, cancel := context.WithCancel(context.Background())
	wd.Go(ctx, thelp.NonCompliantTask)
	wd.Go(ctx, thelp.NonCompliantTask)
	wd.Go(ctx, thelp.NonCompliantTask)
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

	ow := thelp.NewOwnerFake()
	wd := wdog.New(wdog.NewConfiguration(ow))

	wd.Watch()
	defer wd.Close()

	ctx, cancel := context.WithCancel(context.Background())
	wd.Go(ctx, thelp.CompliantTask)
	wd.Go(ctx, thelp.CompliantTask)
	wd.Go(ctx, thelp.CompliantTask)
	cancel()

	// Teardown time
	time.Sleep(time.Millisecond * 500)

	if ow.LastNoise() != wdog.Silence {
		t.Error("Unexpected noise value")
	}
}

func TestPanicTaskMultipleTimes(t *testing.T) {
	t.Parallel()

	ow := thelp.NewOwnerFake()
	wd := wdog.New(wdog.NewConfiguration(ow))

	wd.Watch()
	defer wd.Close()

	ctx, cancel := context.WithCancel(context.Background())
	wd.Go(ctx, thelp.PanicTask)
	wd.Go(ctx, thelp.PanicTask)
	wd.Go(ctx, thelp.PanicTask)
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
