package mbvip

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"testing/fstest"
	"time"
)

func TestIframePolicy(t *testing.T) {
	a := &Assets{}
	if got := a.csp(); got != assetCSP+"'none'" {
		t.Fatal(got)
	}
	a.allowIframes = true
	if got := a.csp(); got != assetCSP+"http: https:" {
		t.Fatal(got)
	}
}

func TestNativeIframes(t *testing.T) {
	if os.Getenv("MINI_ELECTRON_TEST_IFRAMES") != "1" {
		t.Skip("requires an interactive Windows desktop")
	}
	newServer := func(start func(http.Handler) *httptest.Server) (*httptest.Server, <-chan struct{}) {
		loaded := make(chan struct{}, 1)
		server := start(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/ready" {
				select {
				case loaded <- struct{}{}:
				default:
				}
				return
			}
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<script>fetch('/ready')</script>`)
		}))
		t.Cleanup(server.Close)
		return server, loaded
	}
	plain, httpReady := newServer(httptest.NewServer)
	secure, httpsReady := newServer(httptest.NewTLSServer)
	dir := t.TempDir()
	t.Setenv("LOCALAPPDATA", dir)
	t.Setenv("MINI_ELECTRON_URL", "")
	config := dir + "/mini-electron.json"
	if err := os.WriteFile(config, []byte(`{"allowIframes":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	a, err := New(Options{ConfigPath: config, DLLPath: "../runtime/mb132_x64.dll", AssetsRoot: ".", Assets: fstest.MapFS{
		"index.html": {Data: fmt.Appendf(nil, `<iframe src="%s"></iframe><iframe src="%s"></iframe>`, plain.URL, secure.URL)},
	}})
	if err != nil {
		t.Fatal(err)
	}
	result := make(chan string, 1)
	go func() {
		deadline := time.NewTimer(10 * time.Second)
		defer deadline.Stop()
		for httpReady != nil || httpsReady != nil {
			select {
			case <-httpReady:
				httpReady = nil
			case <-httpsReady:
				httpsReady = nil
			case <-deadline.C:
				result <- fmt.Sprintf("iframe load timed out (HTTP loaded: %t, HTTPS loaded: %t)", httpReady == nil, httpsReady == nil)
				_ = a.client.PostToUIThread(a.client.exitMessageLoop)
				return
			}
		}
		result <- ""
		_ = a.client.PostToUIThread(a.client.exitMessageLoop)
	}()
	if err := a.Run(); err != nil {
		t.Fatal(err)
	}
	select {
	case failure := <-result:
		if failure != "" {
			t.Fatal(failure)
		}
	default:
		t.Fatal("app exited before iframes loaded")
	}
}
