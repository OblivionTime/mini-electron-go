package mbvip

import (
	"os"
	"testing"
)

func TestAppTrust(t *testing.T) {
	a := App{base: "http://localhost:5173/"}
	for _, tc := range []struct {
		url  string
		want bool
	}{
		{"http://LOCALHOST:5173/settings", true},
		{"https://localhost:5173/", false},
		{"http://localhost:5174/", false},
		{"http://localhost:5173.example.org/", false},
		{"http://user@localhost:5173/", false},
		{"about:blank", false},
	} {
		if got := a.trusted(tc.url); got != tc.want {
			t.Errorf("trusted(%q) = %t, want %t", tc.url, got, tc.want)
		}
	}
	if !loopbackURL("http://[::1]:5173/") || loopbackURL("https://example.org/") {
		t.Fatal("incorrect development URL policy")
	}
	if tokenMatches("", "") || tokenMatches("token", "other") || !tokenMatches("token", "token") {
		t.Fatal("incorrect RPC token policy")
	}
}

func TestMaterializeDLLRepairsSameSizeCorruption(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	data := []byte("engine")
	p, err := materializeDLL(data)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := materializeDLL(data); got != p || err != nil {
		t.Fatalf("repair returned %q, %v", got, err)
	}
	if got, err := os.ReadFile(p); string(got) != string(data) || err != nil {
		t.Fatalf("repaired file = %q, %v", got, err)
	}
}
