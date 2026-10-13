//go:build windows

package mbvip

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// Guards against the case-mismatched export names that silently broke JS
// interaction (mbRunJS vs mbRunJs, mbToDouble vs mbJsToDouble, ...).
func TestDLLProcNames(t *testing.T) {
	dll := ""
	for _, c := range []string{
		filepath.Join("..", "runtime", "mb132_x64.dll"),
		filepath.Join("..", "mb132_x64.dll"),
	} {
		if _, err := os.Stat(c); err == nil {
			dll = c
			break
		}
	}
	if dll == "" {
		t.Skip("mb132_x64.dll not found (looked in ../runtime and ..)")
	}
	f, err := loadDLL(dll)
	if err != nil {
		t.Fatalf("loadDLL: %v", err)
	}
	procs := map[string]*syscall.LazyProc{
		"mbRunJs":                          f.mbRunJS,
		"mbJsToDouble":                     f.mbToDouble,
		"mbJsToString":                     f.mbToString,
		"mbGetCookie":                      f.mbGetCookieW,
		"mbDestroyWebView":                 f.mbDestroyWebWindow,
		"mbNetGetRawHttpHeadInBlinkThread": f.mbNetGetRawHttpHead,
		"mbPostToUiThread":                 f.mbPostToUiThread,
		"mbSetDiskCachePath":               f.mbSetDiskCachePath,
		"mbSetDiskCacheLimitDisk":          f.mbSetDiskCacheLimitDisk,
	}
	for name, p := range procs {
		if err := p.Find(); err != nil {
			t.Errorf("proc %s not found in DLL", name)
		}
	}
}
