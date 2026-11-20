//go:build windows

package mbvip

import (
	"io/fs"
	neturl "net/url"
	"path"
	"strings"
)

const assetCSP = "default-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; object-src 'none'; base-uri 'none'; frame-src "

// Assets serves an embedded frontend (e.g. a Vite build) over a virtual host,
// so the app ships as a single binary with no local HTTP server — like Wails.
type Assets struct {
	client       *Client
	fsys         fs.FS
	host         string
	allowIframes bool
}

func (a *Assets) csp() string {
	if a.allowIframes {
		return assetCSP + "http: https:"
	}
	return assetCSP + "'none'"
}

func NewAssets(c *Client, fsys fs.FS, host string) *Assets {
	return &Assets{client: c, fsys: fsys, host: host}
}

// URL is the entry point to load (e.g. http://app.local/index.html).
func (a *Assets) URL() string {
	return "http://" + a.host + "/index.html"
}

// BaseURL is the base used for relative asset paths.
func (a *Assets) BaseURL() string {
	return "http://" + a.host + "/"
}

// Index returns the entry document, to be loaded via LoadHTMLWithBaseURL.
func (a *Assets) Index() (string, error) {
	b, err := fs.ReadFile(a.fsys, "index.html")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Handler intercepts network requests via mbOnLoadUrlBegin and answers them
// from the embedded FS. Register it before LoadURL.
func (a *Assets) Handler() LoadURLBeginHandler {
	cancel := a.client.dll.dll.NewProc("mbNetCancelRequest")
	return func(_ WebView, url string, job uintptr) bool {
		u, err := neturl.Parse(url)
		if err != nil || origin(url) != origin(a.BaseURL()) {
			if err == nil && strings.EqualFold(u.Hostname(), a.host) && cancel.Find() == nil {
				cancel.Call(job)
				return true
			}
			return false
		}
		name, ok := a.resolve(url)
		data, err := fs.ReadFile(a.fsys, name)
		if !ok || err != nil {
			if cancel.Find() == nil {
				cancel.Call(job)
			} else {
				_ = a.client.NetSetData(job, []byte("not found"))
			}
			return true
		}
		_ = a.client.NetSetHTTPHeaderField(job, "Content-Security-Policy", a.csp(), true)
		mime := mimeFor(name)
		_ = a.client.NetSetHTTPHeaderField(job, "Content-Type", mime, true)
		_ = a.client.NetSetMIMEType(job, mime)
		_ = a.client.NetSetData(job, data)
		return true
	}
}

func (a *Assets) resolve(rawURL string) (string, bool) {
	u, err := neturl.Parse(rawURL)
	if err != nil || origin(rawURL) != origin(a.BaseURL()) {
		return "", false
	}
	name := strings.TrimPrefix(u.Path, "/")
	if name == "" {
		name = "index.html"
	}
	if a.exists(name) {
		return name, true
	}
	// SPA fallback: extensionless path -> index.html
	if path.Ext(name) == "" && a.exists("index.html") {
		return "index.html", true
	}
	return "", false
}

func (a *Assets) exists(name string) bool {
	fi, err := fs.Stat(a.fsys, name)
	return err == nil && !fi.IsDir()
}

func mimeFor(name string) string {
	switch strings.ToLower(path.Ext(name)) {
	case ".html", ".htm":
		return "text/html"
	case ".js", ".mjs":
		return "text/javascript"
	case ".css":
		return "text/css"
	case ".json", ".map":
		return "application/json"
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".ico":
		return "image/x-icon"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	case ".wasm":
		return "application/wasm"
	default:
		return "application/octet-stream"
	}
}
