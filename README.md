# 🐶 wdog

`wdog` is a lightweight, concurrency-safe watchdog for Go, built to monitor goroutines for panics, context leaks, and error bursts. It helps enforce lifecycle discipline in background tasks that can’t afford to misbehave.

---

## 🧠 Why Watchdog?

In concurrent Go programs, goroutines can silently leak after their context is canceled — often while holding expensive resources like DB connections, file handles, or sockets.

`wdog` lets you **supervise** those goroutines. If they panic, ignore context, or error repeatedly, `wdog` emits structured alerts — so you can respond before the system rots from the inside.

But there’s a trade-off.

---

## ⚠️ Disclaimer

**It’s cheaper to fix your code than to monitor it.**
But if you can’t touch it — legacy code, 3rd-party libraries, async callbacks from the depths of hell — **then you probably shouldn’t trust it either**.

`wdog` is not a silver bullet. It’s a leash.
Use it when you can't fix the dog, but still need to stop it from chewing the server.

---

## ⚖️ Performance Trade-off

Each `wdog` instance spawns:

* **1 goroutine** to listen to internal events.
* **1 goroutine** to track error tolerance.

Don’t create multiple `wdog` instances unless absolutely necessary — it's designed to be shared.

Additionally, for **each goroutine you register**, `wdog` spins up:

* **1 supervisor goroutine** to monitor the context.
* **1 timer goroutine** to enforce the teardown timeout.

That’s **\~2 goroutines per watched goroutine.**

Is that worth it?

> ✅ **Yes** — if the goroutine is holding a scarce or expensive resource.

Examples:

* DB connections from a pool
* Persistent sockets (WebSocket, gRPC)
* Mutexes, semaphores, or large memory regions

In these cases, the **cost of leaking** outweighs the cost of watching.

> ❌ **No** — if the goroutine is cheap, short-lived, and stateless. `wdog` would just add overhead.

---

## ✅ When to Use

Use `wdog` if:

* Your goroutines might stall or leak.
* The cost of leaked resources is significant.
* You want structured, observable supervision.

Avoid it if:

* You’re spawning high-frequency, trivial workers.
* Overhead is more critical than accountability.

---

## 📦 Features

* Detects and reports **panics** (`Cry`)
* Emits alerts on **context timeout violations** (`Growl`)
* Monitors **error burst tolerance** (`Bark`)
* Sends structured alerts called **Noises**
* Plug in your own alert handler via the `Owner` interface
* Low-overhead internals with configurable buffers and timeouts
* Optional debug logging for development

---

## 💾 Installation

```bash
go get github.com/DiogoJunqueiraGeraldo/wdog
```

---

## 🚀 Quick Start

```go
package main

import (
    "fmt"
    "sync"

    "github.com/DiogoJunqueiraGeraldo/wdog"
)

var (
    instance *wdog.WDog
    once     sync.Once
)

type LoggerOwner struct{}

func (LoggerOwner) Hear(n wdog.Noise) {
    fmt.Printf("Alert: Type=%s, Error=%v, ErrCount=%d\n", n.Type, n.Error, n.ErrCount)

    if n.Type == wdog.Bark {
        panic("watchdog detected possible resource leak")
    }
}

func WatchDog() *wdog.WDog {
    once.Do(func() {
        cfg := wdog.NewConfiguration(LoggerOwner{})
        wd := wdog.New(cfg)
        wd.Watch()
        instance = wd
    })

    return instance
}
```

---

## 📖 Concepts

* **Noise**: Structured alert emitted by the watchdog (`Cry`, `Growl`, `Bark`).
* **Owner**: Your handler for receiving alerts (`Hear()` method).
* **Tolerance Window/Cap**: If error count within a window exceeds the cap, `wdog` emits a `Bark`.
* **Teardown Timeout**: Max time a task has to stop after context cancelation. Exceed it and you get a `Growl`.

---

## 🤝 Contributing

Contributions, bug reports, and feature ideas are welcome. Stick to existing code style, write tests, and avoid bikeshedding.