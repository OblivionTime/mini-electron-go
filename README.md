# mini_electron

基于 **MiniBlink（mb VIP）** 的 Go 封装，并在其上提供了一套类似 **Wails** 的应用层：

- 一个 `Options` 结构体搞定初始化（窗口、绑定、嵌入前端、事件、多窗口）
- Go 方法自动暴露给 JS：`invoke('App.Method', ...args)`
- Go → JS 事件推送：`on('event', fn)`
- 前端 `embed` 进二进制，单 exe、无需本地服务器
- JSON 配置（图标 / exe 名 / 窗口 / 数据目录…）

> 仅支持 **Windows**。`mb*` API 为 MiniBlink **VIP 版**，商用需自行取得授权。

## 目录结构

```
mini_electron/
├─ main.go                 # 应用入口（只有业务逻辑）
├─ mini-electron.json      # 应用配置
├─ build.ps1               # 一键构建
├─ build/icon.ico          # 图标
├─ frontend/               # 前端（Vite + Vue3）
│  ├─ index.html
│  ├─ vite.config.js
│  ├─ package.json
│  └─ src/{main.js, App.vue, bridge.js}
├─ runtime/                # MiniBlink 运行时
│  └─ mb132_x64.dll
├─ mbvip/                  # 封装库（package mbvip）
│  ├─ app_windows.go       # App / Options / 配置（Wails 风格入口）
│  ├─ bind_windows.go      # Binder：Go <-> JS RPC
│  ├─ assets_windows.go    # Assets：嵌入前端资源
│  └─ client/window/page/state/js/...  # 底层 mb API 封装
└─ go.mod
```

> `runtime/` 存放引擎 DLL。MiniBlink 会在 DLL 所在目录生成一个空的 `cookies.dat`，
> 放在这里就不会污染项目根目录。

## 快速开始

```powershell
cd frontend
npm install
npm run build

cd ..
go run .            # 或 ./build.ps1 生成 exe
```

开发模式（Vite 热更新）：

```powershell
npm run dev                      # 终端1（frontend/）
$env:MINI_ELECTRON_DEV="1"; go run .   # 终端2
```

---

# API

## 1. 最简应用

`main.go` 只需要一个 `Options` 结构体：

```go
package main

import (
	"embed"
	"mini_electron/mbvip"
)

//go:embed all:frontend/dist
var frontend embed.FS

func main() {
	mbvip.Run(mbvip.Options{
		Assets:     frontend,
		AssetsRoot: "frontend/dist",
		Bind:       []mbvip.Binding{{Name: "App", Target: &Backend{}}},
	})
}
```

`mbvip.New(opts)` 只初始化不阻塞，返回 `*mbvip.App`；`mbvip.Run(opts)` = `New` + `Run()`。

`Options` 字段：

| 字段 | 说明 |
|---|---|
| `Assets` | 嵌入的前端（`embed.FS`）；为 nil 则用 `DevURL` |
| `AssetsRoot` | `Assets` 内子目录（默认 `frontend/dist`） |
| `Host` | 虚拟域名（默认 `app.local`） |
| `URL` | 直接用某个 URL（覆盖一切，等价 dev） |
| `Dev` | 用配置里的 `DevURL`（Vite dev server） |
| `Bind` | `[]Binding{{Name, Target}}`，暴露给 JS |
| `ConfigPath` | 配置文件路径（默认自动查找） |
| `DLLPath` | 引擎 DLL 路径（默认自动查找 `runtime/mb132_x64.dll`） |

环境变量覆盖（无需在 main 里写 flag）：
`MINI_ELECTRON_CONFIG` / `MINI_ELECTRON_DLL` / `MINI_ELECTRON_URL` / `MINI_ELECTRON_DEV`。

## 2. 后端结构体（绑定 + 事件 + 多窗口）

导出方法自动变成 `App.Method`。实现 `SetApp(*mbvip.App)` 即可拿到 `App`，
用来 `Emit`（推事件）或 `OpenWindow`（开新窗口）：

```go
type Backend struct {
	app  *mbvip.App
	stop chan struct{}
}

func (b *Backend) SetApp(app *mbvip.App) { b.app = app }

func (b *Backend) Greet(name string) string { return "Hello, " + name }
func (b *Backend) Add(x, y int) int        { return x + y }
func (b *Backend) Save(s string) error     { return nil }   // 支持 (T, error)

func (b *Backend) OpenWindow() { _, _ = b.app.OpenWindow("") }

func (b *Backend) StartTicker() {
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
				b.app.Emit("tick", map[string]any{"count": i}) // 广播到所有窗口
			}
		}
	}()
}
```

`App` 的方法：

| 方法 | 说明 |
|---|---|
| `Run()` | 打开首个窗口并进入消息循环（阻塞） |
| `Bind(name, target)` | 追加绑定（也应用到已开窗口） |
| `Emit(event, data)` | 向所有窗口推事件（任意 goroutine 安全） |
| `OpenWindow(url)` | 新开窗口；`url==""` 加载应用自身 |
| `Client()` | 取底层 `*mbvip.Client`（高级用法） |
| `Config()` | 取加载后的配置 |

## 3. 前端 `bridge.js`

```js
// JS -> Go（Promise）
const n = await invoke('App.Add', 2, 40)      // 42
const s = await invoke('App.Greet', 'Bob')    // "Hello, Bob"

// Go -> JS 事件
on('tick', (data) => console.log(data.count))
```

`invoke` 走 MiniBlink 的 `window.mbQuery`，由 Go 侧 `mbResponseQuery` 应答。

## 4. 配置文件 `mini-electron.json`

放在 exe 同目录或当前目录（`MINI_ELECTRON_CONFIG` 可指定）。

| 字段 | 说明 |
|---|---|
| `name` | 应用名（日志/默认标题） |
| `outputFilename` | 构建产物 exe 名（`build.ps1` 用） |
| `window.title/width/height` | 窗口标题与尺寸 |
| `icon` | `.ico` 路径（相对配置文件）→ 窗口/任务栏图标 |
| `dataDir` | Cookie/缓存/localStorage 目录（`%LOCALAPPDATA%` 下） |
| `cookieDir` | Cookie 存放位置（可选）。空 = 默认 `<dataDir>/cookies`；相对路径相对配置文件 |
| `userAgent` | 自定义 UA |
| `devUrl` | 开发模式地址 |

```json
{
  "name": "Mini Electron Demo",
  "outputFilename": "mini_electron",
  "window": { "title": "Mini Electron Demo", "width": 900, "height": 640 },
  "icon": "build/icon.ico",
  "dataDir": "mini-electron-demo",
  "cookieDir": "",
  "userAgent": "",
  "devUrl": "http://localhost:5173"
}
```

## 5. 构建

```powershell
./build.ps1                 # 前端 + exe
./build.ps1 -SkipFrontend   # 只重编 Go
```

脚本：读配置取 `outputFilename` → 构建前端 → （可选）用 `goversioninfo` 嵌入 exe 图标 →
`go build -ldflags "-H=windowsgui -s -w"`（**GUI 子系统，无黑控制台**）。

exe 图标需 `goversioninfo`（`go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest`）；
没装则跳过，窗口/任务栏图标仍由运行时设置生效。

---

# 底层 API（高级）

需要更细粒度控制时可直接用 `*mbvip.Client`：

```go
client, _ := mbvip.NewClient(mbvip.Config{DLLPath: "runtime/mb132_x64.dll"})
client.Init()
view, _ := client.CreateWebWindow(mbvip.WindowOptions{Width: 900, Height: 640})

// 缓存（必须在 LoadURL 之前）
client.SetCookieJarPath(view, dir+"/cookies")
client.SetLocalStorageFullPath(view, dir+"/localstorage")
client.SetDiskCachePath(view, dir+"/cache")
client.SetDiskCacheEnabled(view, true)

// 导航 / JS / 窗口
client.LoadURL(view, "http://x.com")
client.LoadHTMLWithBaseURL(view, html, "http://app.local/")
client.RunJSMainFrame(view, "console.log('hi')")
client.SetWindowTitle(view, "标题")
client.SetUserAgent(view, "MyAgent/1.0")
mbvip.SetWindowIconFromFile(hwnd, "icon.ico")

// 事件回调
client.OnTitleChanged(view, func(v mbvip.WebView, t string) {})
client.OnDocumentReady(view, func(v mbvip.WebView, f mbvip.WebFrameHandle) {})
client.OnLoadURLBegin(view, func(v mbvip.WebView, url string, job uintptr) bool { return false })
client.OnCreateView(view, func(v mbvip.WebView, t int32, url string, _ uintptr) mbvip.WebView { return 0 })
client.OnClose(view, func(v mbvip.WebView) bool { return true })
client.OnDestroy(view, func(v mbvip.WebView) {})

client.PostToUIThread(func() { ... }) // 任意 goroutine 回到 UI 线程
client.RunMessageLoop()
```

手动用 `Binder` / `Assets`：

```go
binder := mbvip.NewBinder(client, view)
binder.Bind("App", &Backend{}); binder.Install()
binder.Emit("tick", data)

assets := mbvip.NewAssets(client, distFS, "app.local")
html, _ := assets.Index()
client.OnLoadURLBegin(view, assets.Handler())
client.LoadHTMLWithBaseURL(view, html, assets.BaseURL())
```

---

## 注意事项

- **仅 Windows**；`mb*` API 为 MiniBlink **VIP** 版。
- **线程模型**：`mbInit` 及其后 API 必须在同一线程调用；跨线程操作 UI/JS 用
  `client.PostToUIThread(func(){...})`（`App.Emit` 已内部处理）。
- 缓存/cookie/localStorage 路径要在 `LoadURL` **之前**设置（框架已处理）。
- `<script type="module">` 需响应带 `Content-Type`；`Assets` 已处理。
- 发布用 `-H=windowsgui` 去掉黑色控制台；启动时也会自动隐藏独占的控制台。
- 引擎 DLL 放在 `runtime/`；MiniBlink 会在此目录生成空的 `cookies.dat`（正常现象）。
- Cookie 默认存在 `<dataDir>/cookies`；要改位置在配置里填 `cookieDir`（空则用默认）。
