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

开发模式（Vite 热更新）：把 `mini-electron.json` 里的 `debug` 设为 `true`，
然后启动 Vite：

```powershell
npm run dev                      # 终端1（frontend/）
# 终端2：确认 mini-electron.json 里 "debug": true，然后
go run .
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
| `URL` | 指定本机回环 HTTP(S) 开发地址，覆盖环境变量和配置 |
| `Dev` | 用配置里的 `DevURL`（Vite dev server） |
| `Bind` | `[]Binding{{Name, Target}}`，暴露给 JS |
| `ConfigPath` | 配置文件路径（默认自动查找） |
| `DLLPath` | 引擎 DLL 路径（默认自动查找 `runtime/mb132_x64.dll`） |
| `DLLData` | 内嵌的引擎 DLL 字节；启动时释放到用户缓存目录再加载（打包单 exe 用） |

环境变量覆盖：`MINI_ELECTRON_CONFIG` / `MINI_ELECTRON_DLL` / `MINI_ELECTRON_URL`。
调试模式由配置文件的 `debug` 字段控制（见下），不再依赖环境变量。

## 2. 后端结构体（绑定 + 事件 + 多窗口）

导出方法自动变成 `App.Method`。`SetApp(*mbvip.App)` 只供框架注入，不对 JS 暴露；绑定不支持可变参数方法。实现 `SetApp` 即可拿到 `App`，
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
| `OpenWindow(url)` | 空字符串加载应用自身；同源 URL 新开应用窗口；外部 HTTP(S) 交给系统浏览器，返回零 `WebView` |
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
| `debug` | `true` = 调试模式，加载 `devUrl` 而不是内嵌前端；`false` = 加载内嵌前端 |
| `allowIframes` | 默认 `false`；设为 `true` 允许内嵌前端加载任意来源的 HTTP(S) iframe |
| `dll` | 引擎 DLL 路径覆盖（可选，相对配置文件）；空 = 使用内嵌 DLL，未内嵌时才自动查找 |
| `sign` | exe 签名配置（仅供 `build.ps1` 使用，见「构建」） |

```json
{
  "name": "Mini Electron Demo",
  "outputFilename": "mini_electron",
  "window": { "title": "Mini Electron Demo", "width": 900, "height": 640, "maximized": false },
  "icon": "build/icon.ico",
  "dataDir": "mini-electron-demo",
  "cookieDir": "",
  "userAgent": "",
  "devUrl": "http://localhost:5173",
  "debug": false,
  "dll": ""
}
```

完整示例见 `mini-electron.example.json`。

## 5. 调试前端

用**浏览器 DevTools**（最完整、带热更新）：把 `debug` 设为 `true` 启动 Vite dev server，
然后在 Chrome/Edge 打开 `http://localhost:5173` 调试；`go run .` 的窗口也会加载该地址，两边同步。

```json
{ "debug": true, "devUrl": "http://localhost:5173" }
```
```powershell
cd frontend; npm run dev     # 终端1
go run .                     # 终端2
```

> 说明：Miniblink 原生 devtools 在 49/57 内核可用（`mbSetDebugConfig(view,"showDevTools",...)`），
> 但当前使用的 `mb132_x64.dll`（Chromium 132 内核）**尚未支持原生 devtools**，
> 所以应用内调试请走上面的浏览器方式。

## 6. 生成前端调用函数（Wails 风格 bindgen）

绑定到 `Options.Bind` 的 Go 结构体，可以自动生成对应的前端调用函数，省去手写
`invoke('App.Method', ...)`：

```powershell
go run . -gen frontend/src/bindings   # 生成到此目录（默认就是它）
```

会按绑定名生成文件：

```text
frontend/src/bindings/
├─ App.js        # export function Greet(arg1) { return window.__mbInvoke('App.Greet', arg1) }
├─ App.d.ts      # export function Greet(arg1: string): Promise<string>
├─ index.js      # export * as App from './App.js'
└─ index.d.ts
```

用法：

```js
import { App } from './bindings'          // 或 import { Greet } from './bindings/App'
const greeting = await App.Greet('Bob')   // 等价于 invoke('App.Greet', 'Bob')
```

规则（与运行时 `Binder` 一致）：

- 只生成导出方法（跳过框架钩子 `SetApp`）。
- 参数名统一为 `arg1, arg2…`（Go 反射拿不到形参名，Wails 亦如此）。
- 类型映射：`string`→`string`，数字→`number`，`bool`→`boolean`，`[]byte`→`string`（base64），
  切片/数组→`T[]`，map→`Record<K,V>`，`time.Time`→`string`，其余结构体→`any`。
- 返回值：无→`void`，有 `error` 会被 reject，多返回值→元组 `[A, B]`。
- 生成的代码依赖 `bridge.js` 暴露的全局 `window.__mbInvoke`（`bridge.js` 会自动设置），
  所以应用里要先引入 `bridge.js`（模板的 `App.vue` 已经在引）。

生成由 `build.ps1` 在**前端构建之前**自动执行（可用 `-SkipGen` 跳过），
生成目录建议一并提交，方便类型提示。

## 7. 构建

```powershell
./build.ps1                 # 生成绑定 + 前端 + exe（+ 签名，若已配置）
./build.ps1 -SkipFrontend   # 只重编 Go
./build.ps1 -SkipGen        # 跳过绑定生成
./build.ps1 -SkipSign       # 跳过签名
./build.ps1 -Offline        # 签名时不打时间戳（断网构建）
```

脚本：读配置取 `outputFilename` → 构建前端 → 用 `rsrc` 把图标写进 exe 资源段 →
`go build -ldflags "-H=windowsgui -s -w"`（**GUI 子系统，无黑控制台**）→ （可选）`signtool` 签名。

引擎 DLL 通过 `//go:embed` 打进 exe（`main.go` 里 `//go:embed runtime/mb132_x64.dll`），
首次运行释放到 `%LOCALAPPDATA%` 缓存目录再加载，因此发布时 **无需附带 `runtime/mb132_x64.dll`**。
代价是 exe 体积增加约 DLL 大小；若想外置，删掉该 embed 并让 `findDLL` 查找同目录 DLL 即可。

DLL 优先级：`Options.DLLPath` → `MINI_ELECTRON_DLL` → JSON `dll` → 内嵌 DLL → 自动查找。只使用可信配置和 DLL；配置指定的 DLL 会以本机代码运行。

桥接只给内嵌页面或选定的本机回环开发源；Vite 不要监听或加载不可信内容。桥接凭据按主文档签发，不隔离同源脚本。直接使用底层 `Binder` 不会自动套用 `App` 的策略。

要嵌入外部站点，在 EXE 同目录的 `mini-electron.json` 设置 `"allowIframes": true` 后重启。目标站自己的 `X-Frame-Options` / `frame-ancestors` 仍然有效；建议给外部 iframe 加 `sandbox`，只开放所需权限。当前引擎的导航回调不区分主框架和子框架，开启后也允许程序触发的外站导航，但不会向外站发放 Go 桥接凭据。外站链接点击仍交给系统浏览器。

随包的 `mb132_x64.dll` 已实测可加载本机自签名 HTTPS。`allowIframes` 不改变 TLS 设置，也不为替换的 DLL 保证同样行为；不要用这份引擎处理需要可靠证书验证的登录或敏感数据。

在交互式 Windows 桌面执行 `./smoke.ps1 -Exe <构建产物>`。脚本只复制 EXE，分别用空缓存和已有缓存启动，检查生成绑定的成功、错误和关窗退出。发布前在干净的 Windows x64 虚拟机上也跑一次。

发布前按 [MiniBlink 上游](https://github.com/weolar/miniblink49) 和该 DLL 实际发行渠道核对再分发许可、随附声明及第三方许可；内嵌不改变许可义务。跟踪该发行渠道和 Chromium 安全公告；取得兼容补丁后替换 DLL、重新构建、跑冒烟并重新签名。没有可确认的补丁或再分发许可时，不发布该 DLL。

### exe 图标

用 [`rsrc`](https://github.com/akavel/rsrc) 把 `icon` 指向的 `.ico` 编译成 `rsrc.syso`，
`go build` 会自动链接进 exe 的资源段（资源管理器/任务栏图标），等价于 electron 的 rcedit 步骤。
仅需一次安装：`go install github.com/akavel/rsrc@latest`。没装则回退到 `goversioninfo`，两者都没有就跳过。

### 代码签名（signtool，对齐 electron-builder）

`build.ps1` 的签名逻辑照搬 electron-builder 的 `windowsCodeSign.ts`（`computeSignToolArgs` + `sign()`）：

- **默认双重签名** `signingHashAlgorithms: ["sha1","sha256"]`：先 SHA-1，再 SHA-256（第二遍用 `/as` 追加为嵌套签名），
  这样老版本和新版 Windows 都能验证。只填一个就单签。
- **时间戳**：SHA-256/嵌套签名用 RFC3161 `/tr` + `/td sha256`（`rfc3161TimeStampServer`）；
  SHA-1 用传统 `/t`（`timeStampServer`），两者默认 `http://timestamp.digicert.com`。
  设 `ELECTRON_BUILDER_OFFLINE=true` 或加 `-Offline` 则跳过时间戳。
- **证书来源**：`.pfx`/`.p12` 文件（`/f`），或从 Windows 证书存储按 `certificateSubjectName` / `certificateSha1`
  查找（`/sha1 /s [/sm]`）；可选 `additionalCertificateFile` 用 `/ac` 追加交叉证书。
- **工具查找**：`sign.signtoolPath` → `$env:SIGNTOOL_PATH` → PATH → Windows SDK 的 `...\x64\signtool.exe`。
- **失败重试**：锁定 / 时间戳服务器不可达时等待 15s 重试一次（同 electron-builder）。
- **环境变量**（CI 推荐，优先于配置）：
  `WIN_CSC_LINK`/`CSC_LINK`（`.pfx` 路径、`https://` URL 或 base64）与
  `WIN_CSC_KEY_PASSWORD`/`CSC_KEY_PASSWORD`；只要设置了 `CSC_LINK` 就自动签名。
- 签完自动 `signtool verify /pa` 校验；`forceCodeSigning: true` 要求签名及校验均成功，不能与 `-SkipSign` 同用。

`mini-electron.json` 的 `sign` 段：

```json
"sign": {
  "enabled": true,
  "certificateFile": "cert/cert.pfx",
  "certificatePassword": "",
  "certificateSubjectName": "",
  "certificateSha1": "",
  "additionalCertificateFile": "",
  "signingHashAlgorithms": ["sha1", "sha256"],
  "timeStampServer": "http://timestamp.digicert.com",
  "rfc3161TimeStampServer": "http://timestamp.digicert.com",
  "description": "",
  "website": "",
  "signtoolPath": "",
  "forceCodeSigning": false
}
```

`description` 默认取配置的 `name`。EV 证书（私钥在硬件/HSM 里不可导出）用 `certificateSha1`/`certificateSubjectName`
从证书存储选择，而不是 `certificateFile`。

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
