package thelp

import (
	"errors"
	"fmt"
	"github.com/DiogoJunqueiraGeraldo/wdog"
)

type OwnerMock struct {
	NoiseMemory []wdog.Noise
}

func NewOwnerFake() *OwnerMock {
	return &OwnerMock{
		NoiseMemory: make([]wdog.Noise, 0, 1024),
	}
}

func (o *OwnerMock) Hear(noise wdog.Noise) {
	o.NoiseMemory = append(o.NoiseMemory, noise)
}

func (o *OwnerMock) LastNoise() wdog.Noise {
	noiseLen := len(o.NoiseMemory)
	if noiseLen == 0 {
		return wdog.Silence
	}

	return o.NoiseMemory[noiseLen-1]
}

func (o *OwnerMock) DiffHistory(expect []wdog.NoiseType) error {
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
