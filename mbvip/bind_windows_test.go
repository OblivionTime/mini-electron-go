//go:build windows

package mbvip

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

type binderCalc struct{}

func (*binderCalc) Add(a, b int) int      { return a + b }
func (*binderCalc) Name(s string) string  { return "hi " + s }
func (*binderCalc) Boom() (string, error) { return "", errors.New("boom") }
func (*binderCalc) Pair() (int, int)      { return 1, 2 }
func (*binderCalc) Void()                 {}

func TestBinderBind(t *testing.T) {
	b := &Binder{targets: map[string]reflect.Value{}}
	if err := b.Bind("App", &binderCalc{}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"App.Add", "App.Name", "App.Boom", "App.Pair", "App.Void"} {
		if _, ok := b.targets[name]; !ok {
			t.Errorf("method not registered: %s", name)
		}
	}
}

func TestBinderCall(t *testing.T) {
	b := &Binder{targets: map[string]reflect.Value{}}
	mv := reflect.ValueOf(&binderCalc{})

	cases := []struct {
		method string
		args   []string
		want   any
	}{
		{"Add", []string{"2", "40"}, 42},
		{"Name", []string{`"x"`}, "hi x"},
		{"Pair", nil, []any{1, 2}},
		{"Void", nil, nil},
	}
	for _, c := range cases {
		raw := make([]json.RawMessage, len(c.args))
		for i, a := range c.args {
			raw[i] = json.RawMessage(a)
		}
		got, err := b.call(mv.MethodByName(c.method), raw)
		if err != nil {
			t.Fatalf("%s: %v", c.method, err)
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Fatalf("%s: got %#v want %#v", c.method, got, c.want)
		}
	}
}

func TestBinderErrors(t *testing.T) {
	b := &Binder{}
	mv := reflect.ValueOf(&binderCalc{})

	if _, err := b.call(mv.MethodByName("Boom"), nil); err == nil || err.Error() != "boom" {
		t.Fatalf("expected boom, got %v", err)
	}
	if _, err := b.call(mv.MethodByName("Add"), []json.RawMessage{json.RawMessage("1")}); err == nil {
		t.Fatal("expected arg count error")
	}
	if _, err := b.call(mv.MethodByName("Add"), []json.RawMessage{json.RawMessage("1"), json.RawMessage(`"x"`)}); err == nil {
		t.Fatal("expected unmarshal error")
	}
}
