//go:build windows

package mbvip

import (
	"encoding/json"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"unsafe"
)

// WindowConfig mirrors the "window" object in mini-electron.json.
type WindowConfig struct {
	Title  string `json:"title"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// AppConfig mirrors mini-electron.json.
type AppConfig struct {
	Name           string       `json:"name"`
	OutputFilename string       `json:"outputFilename"`
	Window         WindowConfig `json:"window"`
	Icon           string       `json:"icon"`
	DataDir        string       `json:"dataDir"`
	CookieDir      string       `json:"cookieDir"`
	UserAgent      string       `json:"userAgent"`
	DevURL         string       `json:"devUrl"`
}

// Binding exposes a Go struct's methods to JS under "Name.Method".
type Binding struct {
	Name   string
	Target any
}

// Options configures an App.
type Options struct {
	ConfigPath string // mini-electron.json (default: auto-find)
	DLLPath    string // mb132_x64.dll (default: auto-find)
	Assets     fs.FS  // embedded frontend; nil = use dev URL
	AssetsRoot string // subdir inside Assets (default "frontend/dist")
	Host       string // virtual host for Assets (default "app.local")
	URL        string // explicit URL override (implies dev)
	Dev        bool   // load from AppConfig.DevURL instead of Assets
	Bind       []Binding
}

// App is a ready-to-run MiniBlink application: window + bindings + embedded
// assets + events, driven entirely by mini-electron.json.
type App struct {
	opts   Options
	cfg    AppConfig
	client *Client
	assets *Assets
	html   string
	base   string
	dir    string
	cfgDir string

	mu       sync.Mutex
	bindings []Binding
	wins     map[WebView]*Binder
}

// Run builds an App and blocks on the message loop.
func Run(opts Options) {
	app, err := New(opts)
	if err != nil {
		log.Fatalf("mbvip: %v", err)
	}
	if err := app.Run(); err != nil {
		log.Fatalf("mbvip: %v", err)
	}
}

// New creates an App, loads config, initializes MiniBlink and prepares assets.
func New(opts Options) (*App, error) {
	hideConsole()

	cfgPath := opts.ConfigPath
	if cfgPath == "" {
		cfgPath = os.Getenv("MINI_ELECTRON_CONFIG")
	}
	cfgPath = findConfig(cfgPath)
	if cfgPath != "" {
		if abs, err := filepath.Abs(cfgPath); err == nil {
			cfgPath = abs
		}
	}
	cfg := LoadConfig(cfgPath)
	cfgDir := ""
	if cfgPath != "" {
		cfgDir = filepath.Dir(cfgPath)
	}
	if cfg.Icon != "" && cfgDir != "" && !filepath.IsAbs(cfg.Icon) {
		cfg.Icon = filepath.Join(cfgDir, cfg.Icon)
	}

	dllPath := opts.DLLPath
	if dllPath == "" {
		dllPath = os.Getenv("MINI_ELECTRON_DLL")
	}
	if dllPath == "" {
		dllPath = findDLL()
	}
	if abs, err := filepath.Abs(dllPath); err == nil {
		dllPath = abs
	}

	dir := filepath.Join(userCacheDir(), cfg.DataDir)
	for _, sub := range []string{"", "localstorage", "cache"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			return nil, err
		}
	}
	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}

	client, err := NewClient(Config{DLLPath: dllPath})
	if err != nil {
		return nil, err
	}
	client.Init()

	if lf, err := os.Create(filepath.Join(dir, "app.log")); err == nil {
		log.SetOutput(lf)
	}

	a := &App{
		opts:     opts,
		cfg:      cfg,
		cfgDir:   cfgDir,
		client:   client,
		dir:      dir,
		bindings: append([]Binding(nil), opts.Bind...),
		wins:     map[WebView]*Binder{},
	}

	switch {
	case opts.URL != "":
		a.base = opts.URL
	case os.Getenv("MINI_ELECTRON_URL") != "":
		a.base = os.Getenv("MINI_ELECTRON_URL")
	case opts.Dev || os.Getenv("MINI_ELECTRON_DEV") != "":
		a.base = cfg.DevURL
	case opts.Assets != nil:
		root := opts.AssetsRoot
		if root == "" {
			root = "frontend/dist"
		}
		sub, err := fs.Sub(opts.Assets, root)
		if err != nil {
			return nil, err
		}
		host := opts.Host
		if host == "" {
			host = "app.local"
		}
		a.assets = NewAssets(client, sub, host)
		if a.html, err = a.assets.Index(); err != nil {
			return nil, err
		}
		a.base = a.assets.BaseURL()
	default:
		a.base = cfg.DevURL
	}

	for _, b := range a.bindings {
		if s, ok := b.Target.(interface{ SetApp(*App) }); ok {
			s.SetApp(a)
		}
	}
	return a, nil
}

// Run opens the first window and blocks on the message loop.
func (a *App) Run() error {
	log.Printf("%s starting (base %s, data dir %s)", a.cfg.Name, a.base, a.dir)
	if _, err := a.OpenWindow(""); err != nil {
		return err
	}
	return a.client.RunMessageLoop()
}

// Client returns the underlying MiniBlink client (escape hatch).
func (a *App) Client() *Client { return a.client }

// Config returns the loaded application config.
func (a *App) Config() AppConfig { return a.cfg }

// Bind exposes a Go struct's exported methods to JS as "name.Method".
// Call it before Run, or at any time (it also applies to open windows).
func (a *App) Bind(name string, target any) *App {
	a.mu.Lock()
	a.bindings = append(a.bindings, Binding{Name: name, Target: target})
	bs := make([]*Binder, 0, len(a.wins))
	for _, b := range a.wins {
		bs = append(bs, b)
	}
	a.mu.Unlock()

	if s, ok := target.(interface{ SetApp(*App) }); ok {
		s.SetApp(a)
	}
	for _, b := range bs {
		_ = b.Bind(name, target)
	}
	return a
}

// Emit sends an event to every open window. Safe from any goroutine.
func (a *App) Emit(event string, data any) {
	a.mu.Lock()
	bs := make([]*Binder, 0, len(a.wins))
	for _, b := range a.wins {
		bs = append(bs, b)
	}
	a.mu.Unlock()
	for _, b := range bs {
		_ = b.Emit(event, data)
	}
}

// OpenWindow creates a new window; url == "" loads the app itself.
func (a *App) OpenWindow(url string) (WebView, error) {
	view, err := a.client.CreateWebWindow(WindowOptions{
		Type:   WindowTypePopup,
		Width:  a.cfg.Window.Width,
		Height: a.cfg.Window.Height,
	})
	if err != nil {
		return 0, err
	}
	a.client.ShowWindow(view, false)
	a.setupView(view)
	log.Printf("window opened view=%d url=%q", view, url)

	switch {
	case url != "":
		a.client.LoadURL(view, url)
	case a.html != "":
		if err := a.client.LoadHTMLWithBaseURL(view, a.html, a.base); err != nil {
			return view, err
		}
	default:
		a.client.LoadURL(view, a.base)
	}
	return view, nil
}

// cookieDir resolves where cookies are stored: the configured cookieDir if set
// (relative paths are resolved against the config file), otherwise the default
// <dataDir>/cookies.
func (a *App) cookieDir() string {
	d := a.cfg.CookieDir
	if d == "" {
		return filepath.Join(a.dir, "cookies")
	}
	if !filepath.IsAbs(d) && a.cfgDir != "" {
		d = filepath.Join(a.cfgDir, d)
	}
	return d
}

func (a *App) setupView(view WebView) {
	must := func(err error) {
		if err != nil {
			log.Printf("mbvip: %v", err)
		}
	}

	cookieDir := a.cookieDir()
	must(os.MkdirAll(cookieDir, 0o755))
	must(a.client.SetCookieJarPath(view, cookieDir))
	must(a.client.SetLocalStorageFullPath(view, filepath.Join(a.dir, "localstorage")))
	must(a.client.SetDiskCachePath(view, filepath.Join(a.dir, "cache")))
	must(a.client.SetDiskCacheEnabled(view, true))
	must(a.client.SetDiskCacheLimitDisk(view, 256*1024*1024))
	if a.cfg.UserAgent != "" {
		must(a.client.SetUserAgent(view, a.cfg.UserAgent))
	}

	binder := NewBinder(a.client, view)
	for _, b := range a.bindings {
		if err := binder.Bind(b.Name, b.Target); err != nil {
			log.Printf("mbvip: bind %s: %v", b.Name, err)
		}
	}
	if err := binder.Install(); err != nil {
		log.Printf("mbvip: install: %v", err)
	}
	a.mu.Lock()
	a.wins[view] = binder
	a.mu.Unlock()

	if a.assets != nil {
		must(a.client.OnLoadURLBegin(view, a.assets.Handler()))
	}

	// window.open / target=_blank -> a real second window
	must(a.client.OnCreateView(view, func(_ WebView, _ int32, url string, _ uintptr) WebView {
		nv, err := a.OpenWindow(url)
		if err != nil {
			log.Printf("mbvip: create view: %v", err)
			return 0
		}
		return nv
	}))

	// show only once the page is parsed -> no black flash
	must(a.client.OnDocumentReady(view, func(v WebView, _ WebFrameHandle) {
		if a.cfg.Window.Title != "" {
			_ = a.client.SetWindowTitle(v, a.cfg.Window.Title)
		}
		if a.cfg.Icon != "" {
			if _, err := os.Stat(a.cfg.Icon); err == nil {
				if hwnd, err := a.client.HostHWND(v); err == nil {
					if err := SetWindowIconFromFile(hwnd, a.cfg.Icon); err != nil {
						log.Printf("mbvip: set window icon: %v", err)
					}
				}
			}
		}
		a.client.MoveToCenter(v)
		a.client.ShowWindow(v, true)
		a.client.SetFocus(v)
		log.Printf("window ready view=%d", v)
	}))

	must(a.client.OnDestroy(view, func(v WebView) {
		a.mu.Lock()
		delete(a.wins, v)
		a.mu.Unlock()
	}))
}

// LoadConfig reads mini-electron.json, applying defaults for missing fields.
func LoadConfig(path string) AppConfig {
	cfg := AppConfig{
		Name:           "Mini Electron App",
		OutputFilename: "app",
		DataDir:        "mini-electron-app",
		DevURL:         "http://localhost:5173",
	}
	cfg.Window.Title = cfg.Name
	cfg.Window.Width = 900
	cfg.Window.Height = 640

	if path != "" {
		if b, err := os.ReadFile(path); err == nil {
			_ = json.Unmarshal(b, &cfg)
		}
	}
	if cfg.Window.Width <= 0 {
		cfg.Window.Width = 900
	}
	if cfg.Window.Height <= 0 {
		cfg.Window.Height = 640
	}
	if cfg.Window.Title == "" {
		cfg.Window.Title = cfg.Name
	}
	if cfg.DataDir == "" {
		cfg.DataDir = "mini-electron-app"
	}
	if cfg.DevURL == "" {
		cfg.DevURL = "http://localhost:5173"
	}
	return cfg
}

func findConfig(explicit string) string {
	if explicit != "" {
		return explicit
	}
	var candidates []string
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "mini-electron.json"))
	}
	candidates = append(candidates, "mini-electron.json")
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

func findDLL() string {
	candidates := []string{
		filepath.Join("runtime", "mb132_x64.dll"),
		"mb132_x64.dll",
	}
	if exe, err := os.Executable(); err == nil {
		base := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(base, "runtime", "mb132_x64.dll"),
			filepath.Join(base, "mb132_x64.dll"),
		)
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return filepath.Join("runtime", "mb132_x64.dll")
}

func userCacheDir() string {
	if d, err := os.UserCacheDir(); err == nil {
		return d
	}
	return "."
}

func hideConsole() {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	user32 := syscall.NewLazyDLL("user32.dll")
	getConsoleWindow := kernel32.NewProc("GetConsoleWindow")
	getConsoleProcessList := kernel32.NewProc("GetConsoleProcessList")
	showWindow := user32.NewProc("ShowWindow")
	if getConsoleWindow.Find() != nil || getConsoleProcessList.Find() != nil || showWindow.Find() != nil {
		return
	}
	hwnd, _, _ := getConsoleWindow.Call()
	if hwnd == 0 {
		return
	}
	var pids [2]uint32
	n, _, _ := getConsoleProcessList.Call(uintptr(unsafe.Pointer(&pids[0])), uintptr(len(pids)))
	if n != 1 {
		return // console is shared (e.g. `go run` in a terminal) -> leave it
	}
	showWindow.Call(hwnd, 0) // SW_HIDE
}
