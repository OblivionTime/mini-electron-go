//go:build windows

package mbvip

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateBindings(t *testing.T) {
	dir := t.TempDir()
	if err := GenerateBindings(dir, []Binding{{Name: "App", Target: &binderCalc{}}}); err != nil {
		t.Fatal(err)
	}

	js, err := os.ReadFile(filepath.Join(dir, "App.js"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`export function Add(arg1, arg2)`,
		`window.__mbInvoke("App.Add", arg1, arg2)`,
		`export function Void()`,
		`window.__mbInvoke("App.Void")`,
	} {
		if !strings.Contains(string(js), want) {
			t.Errorf("App.js missing %q\n%s", want, js)
		}
	}

	dts, err := os.ReadFile(filepath.Join(dir, "App.d.ts"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`export function Add(arg1: number, arg2: number): Promise<number>`,
		`export function Name(arg1: string): Promise<string>`,
		`export function Boom(): Promise<string>`,
		`export function Pair(): Promise<[number, number]>`,
		`export function Void(): Promise<void>`,
	} {
		if !strings.Contains(string(dts), want) {
			t.Errorf("App.d.ts missing %q\n%s", want, dts)
		}
	}

	idx, err := os.ReadFile(filepath.Join(dir, "index.js"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(idx), `export * as App from './App.js'`) {
		t.Errorf("index.js missing namespace export: %s", idx)
	}
}
