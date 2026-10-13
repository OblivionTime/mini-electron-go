package main

import (
	"embed"
	"time"

	"mini_electron/mbvip"
)

//go:embed all:frontend/dist
var frontend embed.FS

// Backend is exposed to JS as window bridge "App.*".
type Backend struct {
	app  *mbvip.App
	stop chan struct{}
}

// SetApp is called by the framework so the backend can emit events / open windows.
func (b *Backend) SetApp(app *mbvip.App) { b.app = app }

func (b *Backend) Greet(name string) string {
	if name == "" {
		name = "world"
	}
	return "Hello, " + name + " from Go!"
}

func (b *Backend) Add(x, y int) int { return x + y }

func (b *Backend) Now() string { return time.Now().Format("2006-01-02 15:04:05") }

func (b *Backend) OpenWindow() { _, _ = b.app.OpenWindow("") }

func (b *Backend) StartTicker() {
	if b.stop != nil {
		return
	}
	ch := make(chan struct{})
	b.stop = ch
	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()
		for i := 1; ; i++ {
			select {
			case <-ch:
				return
			case <-t.C:
				b.app.Emit("tick", map[string]any{
					"count": i,
					"time":  time.Now().Format("15:04:05"),
				})
			}
		}
	}()
}

func (b *Backend) StopTicker() {
	if b.stop != nil {
		close(b.stop)
		b.stop = nil
	}
}

func main() {
	mbvip.Run(mbvip.Options{
		Assets:     frontend,
		AssetsRoot: "frontend/dist",
		Bind:       []mbvip.Binding{{Name: "App", Target: &Backend{}}},
	})
}
