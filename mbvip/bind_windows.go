//go:build windows

package mbvip

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
)

type binderRequest struct {
	ID     int64             `json:"id"`
	Method string            `json:"method"`
	Args   []json.RawMessage `json:"args"`
	Token  string            `json:"token"`
}

type binderResponse struct {
	ID     int64  `json:"id"`
	Result any    `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

type Binder struct {
	client    *Client
	view      WebView
	mu        sync.RWMutex
	targets   map[string]reflect.Value
	authorize func(JsQuery, string) bool
}

func NewBinder(c *Client, view WebView) *Binder {
	return &Binder{
		client:  c,
		view:    view,
		targets: make(map[string]reflect.Value),
	}
}

func bindingMethods(target any) ([]reflect.Method, error) {
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return nil, fmt.Errorf("mbvip: Bind target must be a non-nil pointer to struct")
	}
	t := v.Type()
	var methods []reflect.Method
	for i := 0; i < t.NumMethod(); i++ {
		m := t.Method(i)
		if m.PkgPath != "" || m.Name == "SetApp" {
			continue
		}
		if m.Type.IsVariadic() {
			return nil, fmt.Errorf("mbvip: variadic method %s is unsupported", m.Name)
		}
		methods = append(methods, m)
	}
	return methods, nil
}

func (b *Binder) Bind(prefix string, target any) error {
	methods, err := bindingMethods(target)
	if err != nil {
		return err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, m := range methods {
		key := m.Name
		if prefix != "" {
			key = prefix + "." + m.Name
		}
		b.targets[key] = reflect.ValueOf(target).Method(m.Index)
	}
	return nil
}

func (b *Binder) Install() error {
	return b.client.OnJsQuery(b.view, b.handleQuery)
}

func (b *Binder) Emit(event string, data any) error {
	ev, err := json.Marshal(event)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	script := fmt.Sprintf("window.__mbEmit && window.__mbEmit(%s, %s)", ev, payload)
	return b.client.PostToUIThread(func() {
		_ = b.client.RunJSMainFrame(b.view, script)
	})
}

func (b *Binder) handleQuery(q JsQuery) {
	res := binderResponse{}
	var req binderRequest
	if err := json.Unmarshal([]byte(q.Request), &req); err != nil {
		res.Error = "invalid request: " + err.Error()
		b.respond(q, res)
		return
	}
	res.ID = req.ID
	if b.authorize != nil && !b.authorize(q, req.Token) {
		res.Error = "bridge unavailable for this document"
		b.respond(q, res)
		return
	}

	b.mu.RLock()
	method, ok := b.targets[req.Method]
	b.mu.RUnlock()
	if !ok {
		res.Error = "unknown method: " + req.Method
		b.respond(q, res)
		return
	}

	result, err := b.call(method, req.Args)
	if err != nil {
		res.Error = err.Error()
	} else {
		res.Result = result
	}
	b.respond(q, res)
}

func tokenMatches(expected, actual string) bool {
	return expected != "" && subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) == 1
}

func (b *Binder) respond(q JsQuery, res binderResponse) {
	data, err := json.Marshal(res)
	if err != nil {
		data = []byte(`{"error":"failed to marshal response"}`)
	}
	_ = q.Respond(string(data))
}

var errType = reflect.TypeOf((*error)(nil)).Elem()

func (b *Binder) call(method reflect.Value, args []json.RawMessage) (any, error) {
	mt := method.Type()
	if len(args) != mt.NumIn() {
		return nil, fmt.Errorf("method expects %d args, got %d", mt.NumIn(), len(args))
	}
	in := make([]reflect.Value, len(args))
	for i, raw := range args {
		p := reflect.New(mt.In(i))
		if err := json.Unmarshal(raw, p.Interface()); err != nil {
			return nil, fmt.Errorf("arg %d: %w", i, err)
		}
		in[i] = p.Elem()
	}
	return splitOut(method.Call(in))
}

func splitOut(out []reflect.Value) (any, error) {
	var callErr error
	n := len(out)
	if n > 0 && out[n-1].Type() == errType {
		if !out[n-1].IsNil() {
			callErr = out[n-1].Interface().(error)
		}
		n--
	}
	switch n {
	case 0:
		return nil, callErr
	case 1:
		return out[0].Interface(), callErr
	default:
		vals := make([]any, n)
		for i := 0; i < n; i++ {
			vals[i] = out[i].Interface()
		}
		return vals, callErr
	}
}
