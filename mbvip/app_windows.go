//go:build windows

package mbvip

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

// WindowConfig mirrors the "window" object in mini-electron.json.
type WindowConfig struct {
	Title     string `json:"title"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Maximized bool   `json:"maximized"`
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
	Debug          bool         `json:"debug"` // load DevURL instead of embedded assets
	AllowIframes   bool         `json:"allowIframes"`
	DLL            string       `json:"dll"` // engine DLL override (relative to config file)
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
	DLLData    []byte // embedded engine DLL; extracted to disk at startup
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
	wins     map[WebView]*appWindow
}

type appWindow struct {
	binder *Binder
	token  string
	hwnd   uintptr
}

// Run builds an App and blocks on the message loop. If launched with
// -gen[=dir] it instead writes the frontend binding wrappers and returns.
func Run(opts Options) {
	if dir, ok := genFlag(os.Args[1:]); ok {
		if err := GenerateBindings(dir, opts.Bind); err != nil {
			log.Fatalf("mbvip: %v", err)
		}
		log.Printf("mbvip: bindings written to %s", dir)
		return
	}
	app, err := New(opts)
	if err != nil {
		log.Fatalf("mbvip: %v", err)
	}
	if err := app.Run(); err != nil {
		log.Fatalf("mbvip: %v", err)
	}
}

// genFlag reports whether args request binding generation and the target dir.
func genFlag(args []string) (string, bool) {
	const def = "frontend/src/bindings"
	for i, a := range args {
		switch {
		case a == "-gen" || a == "--gen" || a == "-mb-gen":
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				return args[i+1], true
			}
			return def, true
		case strings.HasPrefix(a, "-gen="):
			return strings.TrimPrefix(a, "-gen="), true
		case strings.HasPrefix(a, "--gen="):
			return strings.TrimPrefix(a, "--gen="), true
		case strings.HasPrefix(a, "-mb-gen="):
			return strings.TrimPrefix(a, "-mb-gen="), true
		}
	}
	return "", false
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
	if dllPath == "" && cfg.DLL != "" {
		dllPath = cfg.DLL
		if !filepath.IsAbs(dllPath) && cfgDir != "" {
			dllPath = filepath.Join(cfgDir, dllPath)
		}
	}
	if dllPath == "" && len(opts.DLLData) > 0 {
		var err error
		dllPath, err = materializeDLL(opts.DLLData)
		if err != nil {
			return nil, err
		}
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
	a := &App{
		opts:     opts,
		cfg:      cfg,
		cfgDir:   cfgDir,
		client:   client,
		dir:      dir,
		bindings: append([]Binding(nil), opts.Bind...),
		wins:     map[WebView]*appWindow{},
	}

	switch {
	case opts.URL != "":
		a.base = opts.URL
	case os.Getenv("MINI_ELECTRON_URL") != "":
		a.base = os.Getenv("MINI_ELECTRON_URL")
	case opts.Dev || cfg.Debug:
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
		a.assets.allowIframes = cfg.AllowIframes
		if a.html, err = a.assets.Index(); err != nil {
			return nil, err
		}
		a.base = a.assets.BaseURL()
	default:
		a.base = cfg.DevURL
	}
	if a.assets == nil && !loopbackURL(a.base) {
		return nil, fmt.Errorf("mbvip: development URL must be HTTP(S) on loopback")
	}
	if origin(a.base) == "" {
		return nil, fmt.Errorf("mbvip: invalid app URL")
	}
	for _, b := range a.bindings {
		if _, err := bindingMethods(b.Target); err != nil {
			return nil, fmt.Errorf("bind %s: %w", b.Name, err)
		}
	}
	client.Init()

	for _, b := range a.bindings {
		if s, ok := b.Target.(interface{ SetApp(*App) }); ok {
			s.SetApp(a)
		}
	}
	return a, nil
}

// Run opens the first window and blocks on the message loop.
func (a *App) Run() error {
	defer a.client.uninit()
	if lf, err := os.Create(filepath.Join(a.dir, "app.log")); err == nil {
		previous := log.Writer()
		log.SetOutput(lf)
		defer func() { log.SetOutput(previous); _ = lf.Close() }()
	}
	log.Printf("%s starting (base %s, data dir %s)", a.cfg.Name, a.base, a.dir)
	if _, err := a.OpenWindow(""); err != nil {
		return err
	}
	done := make(chan struct{})
	stopped := make(chan struct{})
	go func() { defer close(stopped); a.watchWindows(done) }()
	defer func() { close(done); <-stopped }()
	return a.client.RunMessageLoop()
}

// Client returns the underlying MiniBlink client (escape hatch).
func (a *App) Client() *Client { return a.client }

// Config returns the loaded application config.
func (a *App) Config() AppConfig { return a.cfg }

// Bind exposes a Go struct's exported methods to JS as "name.Method".
// Call it before Run, or at any time (it also applies to open windows).
func (a *App) Bind(name string, target any) *App {
	if _, err := bindingMethods(target); err != nil {
		log.Printf("mbvip: bind %s: %v", name, err)
		return a
	}
	if s, ok := target.(interface{ SetApp(*App) }); ok {
		s.SetApp(a)
	}
	a.mu.Lock()
	a.bindings = append(a.bindings, Binding{Name: name, Target: target})
	bs := make([]*Binder, 0, len(a.wins))
	for _, w := range a.wins {
		bs = append(bs, w.binder)
	}
	a.mu.Unlock()

	for _, b := range bs {
		_ = b.Bind(name, target)
	}
	return a
}

// Emit sends an event to every open window. Safe from any goroutine.
func (a *App) Emit(event string, data any) {
	a.mu.Lock()
	bs := make([]*Binder, 0, len(a.wins))
	for _, w := range a.wins {
		if w.token != "" {
			bs = append(bs, w.binder)
		}
	}
	a.mu.Unlock()
	for _, b := range bs {
		_ = b.Emit(event, data)
	}
}

// OpenWindow creates an app window on the UI thread; an external HTTP(S) URL
// opens in the system browser instead and returns a zero WebView.
func (a *App) OpenWindow(target string) (WebView, error) {
	if target == "" {
		target = a.base
	}
	if !a.trusted(target) {
		return 0, openExternal(target)
	}
	view, err := a.client.CreateWebWindow(WindowOptions{
		Type:   WindowTypePopup,
		Width:  a.cfg.Window.Width,
		Height: a.cfg.Window.Height,
	})
	if err != nil {
		return 0, err
	}
	a.client.ShowWindow(view, false)
	if err := a.setupView(view); err != nil {
		a.client.events.removeView(view)
		_ = a.client.DestroyWebWindow(view)
		return 0, err
	}
	log.Printf("window opened view=%d", view)
	if a.assets != nil && target == a.base {
		// LoadHTMLWithBaseURL has no response headers: establish the policy before any app content.
		html := `<!doctype html><meta http-equiv="Content-Security-Policy" content="` + a.assets.csp() + `">` + a.html
		if err := a.client.LoadHTMLWithBaseURL(view, html, target); err != nil {
			_ = a.client.DestroyWebWindow(view)
			a.client.events.removeView(view)
			a.windowClosed(view)
			return 0, err
		}
		return view, nil
	}
	a.client.LoadURL(view, target)
	return view, nil
}

func origin(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return ""
	}
	return u.Scheme + "://" + strings.ToLower(u.Host)
}

func loopbackURL(raw string) bool {
	if origin(raw) == "" {
		return false
	}
	u, _ := url.Parse(raw)
	host := u.Hostname()
	ip, _ := netip.ParseAddr(host)
	return strings.EqualFold(host, "localhost") || ip.IsLoopback()
}

func (a *App) trusted(raw string) bool {
	got := origin(raw)
	return got != "" && got == origin(a.base)
}

func openExternal(target string) error {
	if origin(target) == "" {
		return fmt.Errorf("mbvip: only HTTP(S) links can be opened externally")
	}
	op := syscall.StringToUTF16Ptr("open")
	arg := syscall.StringToUTF16Ptr(target)
	p := syscall.NewLazyDLL("shell32.dll").NewProc("ShellExecuteW")
	r, _, _ := p.Call(0, uintptr(unsafe.Pointer(op)), uintptr(unsafe.Pointer(arg)), 0, 0, 1)
	if r <= 32 {
		return fmt.Errorf("mbvip: ShellExecuteW failed (%d)", r)
	}
	return nil
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

func (a *App) setupView(view WebView) error {
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

	w := &appWindow{binder: NewBinder(a.client, view)}
	w.binder.authorize = func(q JsQuery, token string) bool {
		a.mu.Lock()
		allowed := q.View == view && a.wins[view] == w && tokenMatches(w.token, token)
		a.mu.Unlock()
		if !allowed {
			return false
		}
		current, err := a.client.GetURL(view)
		return err == nil && a.trusted(current)
	}
	if a.assets != nil {
		if err := a.client.OnLoadURLBegin(view, a.assets.Handler()); err != nil {
			return err
		}
	}
	if err := a.client.OnNavigation(view, func(_ WebView, kind int32, target string) bool {
		// shortcut: this callback has no frame identity, narrow the exception to subframes when MiniBlink exposes it.
		if a.assets != nil && a.assets.allowIframes && (a.trusted(target) || (origin(target) != "" && NavigationType(kind) != NavigationTypeLinkClick)) {
			return true
		}
		a.mu.Lock()
		w.token = ""
		a.mu.Unlock()
		if a.trusted(target) {
			return true
		}
		if NavigationType(kind) == NavigationTypeLinkClick && origin(target) != "" {
			must(a.client.PostToUIThread(func() { must(openExternal(target)) }))
		}
		return false
	}); err != nil {
		return err
	}

	// window.open / target=_blank -> a real second window
	if err := a.client.OnCreateView(view, func(_ WebView, _ int32, target string, _ uintptr) WebView {
		if origin(target) != "" {
			must(a.client.PostToUIThread(func() { _, err := a.OpenWindow(target); must(err) }))
		}
		return 0
	}); err != nil {
		return err
	}

	// show only once the page is parsed -> no black flash
	if err := a.client.OnDocumentReady(view, func(v WebView, frame WebFrameHandle) {
		main, err := a.client.IsMainFrame(v, frame)
		current, urlErr := a.client.GetURL(v)
		if err != nil || !main {
			return
		}
		a.mu.Lock()
		if a.wins[v] != w {
			a.mu.Unlock()
			return
		}
		w.token = ""
		if urlErr != nil || !a.trusted(current) {
			a.mu.Unlock()
			return
		}
		token := rand.Text()
		w.token = token
		if w.hwnd == 0 {
			w.hwnd, _ = a.client.HostHWND(v)
		}
		a.mu.Unlock()
		// shortcut: query callbacks lack a verified frame API, replace the token if the engine exposes one.
		must(a.client.RunJS(v, frame, fmt.Sprintf(`if (window === window.top && location.origin === %q) { window.__mbToken = %q; window.dispatchEvent(new Event('mb-ready')); }`, origin(a.base), token), true))
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
		if a.cfg.Window.Maximized {
			if err := a.client.MaximizeWindow(v); err != nil {
				log.Printf("mbvip: maximize window: %v", err)
			}
		}
		a.client.SetFocus(v)
		log.Printf("window ready view=%d", v)
	}); err != nil {
		return err
	}
	if err := a.client.OnDestroy(view, a.windowClosed); err != nil {
		return err
	}
	if err := w.binder.Install(); err != nil {
		return err
	}
	w.hwnd, _ = a.client.HostHWND(view)
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, b := range a.bindings {
		if err := w.binder.Bind(b.Name, b.Target); err != nil {
			return err
		}
	}
	a.wins[view] = w
	return nil
}

func (a *App) watchWindows(done <-chan struct{}) {
	// shortcut: mb132 can omit destroy callbacks, drop HWND polling when the engine reliably delivers them.
	isWindow := syscall.NewLazyDLL("user32.dll").NewProc("IsWindow")
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			a.mu.Lock()
			var dead []WebView
			for v, w := range a.wins {
				if w.hwnd != 0 {
					if r, _, _ := isWindow.Call(w.hwnd); r == 0 {
						dead = append(dead, v)
					}
				}
			}
			a.mu.Unlock()
			for _, v := range dead {
				a.client.events.removeView(v)
				a.windowClosed(v)
			}
		}
	}
}

// windowClosed 移除已关闭窗口；当没有任何窗口时退出进程，
// 避免关窗后残留后台进程 / 本地服务 / 键盘钩子线程。
func (a *App) windowClosed(v WebView) {
	a.mu.Lock()
	_, existed := a.wins[v]
	delete(a.wins, v)
	remaining := len(a.wins)
	a.mu.Unlock()
	if existed && remaining == 0 {
		if err := a.client.PostToUIThread(func() {
			a.mu.Lock()
			empty := len(a.wins) == 0
			a.mu.Unlock()
			if empty {
				a.client.exitMessageLoop()
			}
		}); err != nil {
			log.Printf("mbvip: exit loop: %v", err)
		}
	}
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

// materializeDLL writes an embedded engine DLL to the user cache dir and
// returns its path. The filename is content-addressed so upgrading the DLL
// never collides with a still-loaded older copy.
func materializeDLL(data []byte) (string, error) {
	sum := sha256.Sum256(data)
	name := "mb132_x64-" + hex.EncodeToString(sum[:]) + ".dll"
	dir := filepath.Join(userCacheDir(), "mini-electron", "runtime")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, name)
	valid := func() bool {
		cached, err := os.ReadFile(path)
		return err == nil && sha256.Sum256(cached) == sum
	}
	if valid() {
		return path, nil
	}
	f, err := os.CreateTemp(dir, "mb132-*.tmp")
	if err != nil {
		return "", err
	}
	defer os.Remove(f.Name())
	_, writeErr := f.Write(data)
	closeErr := f.Close()
	if writeErr != nil {
		return "", writeErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	if err := os.Rename(f.Name(), path); err != nil && !valid() {
		return "", err
	}
	return path, nil
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
