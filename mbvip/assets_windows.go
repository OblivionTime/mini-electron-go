//go:build windows

package mbvip

import (
	"io/fs"
	neturl "net/url"
	"path"
	"strings"
)

// Assets serves an embedded frontend (e.g. a Vite build) over a virtual host,
// so the app ships as a single binary with no local HTTP server — like Wails.
type Assets struct {
	client *Client
	fsys   fs.FS
	host   string
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
	return func(_ WebView, url string, job uintptr) bool {
		name, ok := a.resolve(url)
		if !ok {
			return false
		}
		data, err := fs.ReadFile(a.fsys, name)
		if err != nil {
			return false
		}
		mime := mimeFor(name)
		_ = a.client.NetSetHTTPHeaderField(job, "Content-Type", mime, true)
		_ = a.client.NetSetMIMEType(job, mime)
		_ = a.client.NetSetData(job, data)
		return true
	}
}

func (a *Assets) resolve(rawURL string) (string, bool) {
	u, err := neturl.Parse(rawURL)
	if err != nil || u.Host != a.host {
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
