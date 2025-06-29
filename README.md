# wdog

`wdog` is a lightweight, concurrency-safe watchdog library for Go, designed to monitor goroutines for panics, context violations, and error bursts. It’s ideal for improving resilience and observability in critical background tasks.

---

## Features

- Detects and reports panics in monitored goroutines.
- Alerts when tasks do not respect context cancellation within a configurable teardown timeout.
- Tracks error counts over a sliding tolerance window and emits alerts when thresholds are exceeded.
- Emits structured alerts called **Noises**:
    - **Cry**: task panic
    - **Growl**: context violation (timeout)
    - **Bark**: error tolerance exceeded
- Supports pluggable alert consumers via the `Owner` interface.
- Designed for low overhead with configurable buffer sizes and timeouts.
- Optional debug logging for development and troubleshooting.

---

## Installation

```bash
go get github.com/DiogoJunqueiraGeraldo/wdog
```

## Quick Start

```go
package main

import (
    "fmt"
    "sync"

    "github.com/DiogoJunqueiraGeraldo/wdog"
)

var (
    instance *wdog.WDog
    once sync.Once
)

type LoggerOwner struct{}

func (l LoggerOwner) Hear(n wdog.Noise) {
    fmt.Printf("Received alert: Type=%s, Error=%v, ErrCount=%d\n", n.Type, n.Error, n.ErrCount)
	
    if n.Type == wdog.Bark {
        panic("watchdog notice resource leak")
    }
}

func WatchDog() *wdog.WDog {
    once.Do(func() {
        logger := LoggerOwner{}
        wd := wdog.New(wdog.NewConfiguration(logger))
        wd.Watch()
        instance = wd
    })
    
    return instance
}
```

## Concepts
- **Noise**: A structured alert emitted by the watchdog describing an observed issue.
- **Owner**: The consumer interface for receiving alerts asynchronously.
- **Tolerance Window and Cap**: The watchdog tracks errors over a sliding window; if error count exceeds the cap, a Bark noise is emitted.
- **Teardown Timeout**: Time allowed for a task to gracefully stop after context cancellation; if exceeded, a Growl noise is emitted.


## Contributing
Contributions, issues, and feature requests are welcome. Please adhere to existing code style and write tests where applicable.
