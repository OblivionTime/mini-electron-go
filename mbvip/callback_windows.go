//go:build windows

package mbvip

import (
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"
)

type TitleChangedHandler func(view WebView, title string)
type URLChangedHandler func(view WebView, url string, canGoBack bool, canGoForward bool)
type NavigationHandler func(view WebView, navigationType int32, url string) bool
type DocumentReadyHandler func(view WebView, frameID WebFrameHandle)
type LoadURLBeginHandler func(view WebView, url string, job uintptr) bool
type LoadURLEndHandler func(view WebView, url string, job uintptr, buf uintptr, length int32)
type CreateViewHandler func(view WebView, navigationType int32, url string, windowFeatures uintptr) WebView
type CloseHandler func(view WebView) bool
type DestroyHandler func(view WebView)
type AlertBoxHandler func(view WebView, msg string)
type ConfirmBoxHandler func(view WebView, msg string) bool
type PromptBoxHandler func(view WebView, msg string, defaultResult string) string

type JsQuery struct {
	View      WebView
	ExecState uintptr
	QueryID   int64
	CustomMsg int32
	Request   string
	client    *Client
}

func (q JsQuery) Respond(response string) error {
	if q.client == nil {
		return ProcNotFoundError{Name: "mbResponseQuery"}
	}
	return q.client.ResponseQuery(q.View, q.QueryID, q.CustomMsg, response)
}

type JsQueryHandler func(query JsQuery)

type eventHub struct {
	mu sync.RWMutex

	titleChanged  map[WebView]TitleChangedHandler
	urlChanged    map[WebView]URLChangedHandler
	navigation    map[WebView]NavigationHandler
	documentReady map[WebView]DocumentReadyHandler
	loadURLBegin  map[WebView]LoadURLBeginHandler
	loadURLEnd    map[WebView]LoadURLEndHandler
	jsQuery       map[WebView]JsQueryHandler
	createView    map[WebView]CreateViewHandler
	closeWindow   map[WebView]CloseHandler
	destroyWindow map[WebView]DestroyHandler
	alertBox      map[WebView]AlertBoxHandler
	confirmBox    map[WebView]ConfirmBoxHandler
	promptBox     map[WebView]PromptBoxHandler
}

func newEventHub() *eventHub {
	return &eventHub{
		titleChanged:  make(map[WebView]TitleChangedHandler),
		urlChanged:    make(map[WebView]URLChangedHandler),
		navigation:    make(map[WebView]NavigationHandler),
		documentReady: make(map[WebView]DocumentReadyHandler),
		loadURLBegin:  make(map[WebView]LoadURLBeginHandler),
		loadURLEnd:    make(map[WebView]LoadURLEndHandler),
		jsQuery:       make(map[WebView]JsQueryHandler),
		createView:    make(map[WebView]CreateViewHandler),
		closeWindow:   make(map[WebView]CloseHandler),
		destroyWindow: make(map[WebView]DestroyHandler),
		alertBox:      make(map[WebView]AlertBoxHandler),
		confirmBox:    make(map[WebView]ConfirmBoxHandler),
		promptBox:     make(map[WebView]PromptBoxHandler),
	}
}

func (e *eventHub) setTitleChanged(view WebView, handler TitleChangedHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.titleChanged[view] = handler
}

func (e *eventHub) getTitleChanged(view WebView) TitleChangedHandler {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.titleChanged[view]
}

func (e *eventHub) setURLChanged(view WebView, handler URLChangedHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.urlChanged[view] = handler
}

func (e *eventHub) getURLChanged(view WebView) URLChangedHandler {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.urlChanged[view]
}

func (e *eventHub) setNavigation(view WebView, handler NavigationHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.navigation[view] = handler
}

func (e *eventHub) getNavigation(view WebView) NavigationHandler {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.navigation[view]
}

func (e *eventHub) setDocumentReady(view WebView, handler DocumentReadyHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.documentReady[view] = handler
}

func (e *eventHub) getDocumentReady(view WebView) DocumentReadyHandler {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.documentReady[view]
}

func (e *eventHub) setLoadURLBegin(view WebView, handler LoadURLBeginHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.loadURLBegin[view] = handler
}

func (e *eventHub) getLoadURLBegin(view WebView) LoadURLBeginHandler {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.loadURLBegin[view]
}

func (e *eventHub) setLoadURLEnd(view WebView, handler LoadURLEndHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.loadURLEnd[view] = handler
}

func (e *eventHub) getLoadURLEnd(view WebView) LoadURLEndHandler {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.loadURLEnd[view]
}

func (e *eventHub) setJsQuery(view WebView, handler JsQueryHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.jsQuery[view] = handler
}

func (e *eventHub) getJsQuery(view WebView) JsQueryHandler {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.jsQuery[view]
}

func (e *eventHub) setCreateView(view WebView, handler CreateViewHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.createView[view] = handler
}

func (e *eventHub) getCreateView(view WebView) CreateViewHandler {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.createView[view]
}

func (e *eventHub) setCloseWindow(view WebView, handler CloseHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.closeWindow[view] = handler
}

func (e *eventHub) getCloseWindow(view WebView) CloseHandler {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.closeWindow[view]
}

func (e *eventHub) setDestroyWindow(view WebView, handler DestroyHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.destroyWindow[view] = handler
}

func (e *eventHub) getDestroyWindow(view WebView) DestroyHandler {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.destroyWindow[view]
}

func (e *eventHub) setAlertBox(view WebView, handler AlertBoxHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.alertBox[view] = handler
}

func (e *eventHub) getAlertBox(view WebView) AlertBoxHandler {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.alertBox[view]
}

func (e *eventHub) setConfirmBox(view WebView, handler ConfirmBoxHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.confirmBox[view] = handler
}

func (e *eventHub) getConfirmBox(view WebView) ConfirmBoxHandler {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.confirmBox[view]
}

func (e *eventHub) setPromptBox(view WebView, handler PromptBoxHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.promptBox[view] = handler
}

func (e *eventHub) getPromptBox(view WebView) PromptBoxHandler {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.promptBox[view]
}

func (e *eventHub) removeView(view WebView) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.titleChanged, view)
	delete(e.urlChanged, view)
	delete(e.navigation, view)
	delete(e.documentReady, view)
	delete(e.loadURLBegin, view)
	delete(e.loadURLEnd, view)
	delete(e.jsQuery, view)
	delete(e.createView, view)
	delete(e.closeWindow, view)
	delete(e.destroyWindow, view)
	delete(e.alertBox, view)
	delete(e.confirmBox, view)
	delete(e.promptBox, view)
}

var clientIDSeed atomic.Uintptr
var clientRegistry sync.Map

func nextClientID() uintptr {
	return clientIDSeed.Add(1)
}

func registerClient(client *Client) {
	clientRegistry.Store(client.id, client)
}

func unregisterClient(clientID uintptr) {
	clientRegistry.Delete(clientID)
}

func lookupClient(clientID uintptr) *Client {
	v, ok := clientRegistry.Load(clientID)
	if !ok {
		return nil
	}
	client, ok := v.(*Client)
	if !ok {
		return nil
	}
	return client
}

var titleChangedCallback = syscall.NewCallback(func(webview uintptr, param uintptr, title uintptr) uintptr {
	client := lookupClient(param)
	if client == nil {
		return 0
	}
	handler := client.events.getTitleChanged(WebView(webview))
	if handler == nil {
		return 0
	}
	handler(WebView(webview), utf8FromPtr(title))
	return 0
})

var urlChangedCallback = syscall.NewCallback(func(webview uintptr, param uintptr, url uintptr, canGoBack uintptr, canGoForward uintptr) uintptr {
	client := lookupClient(param)
	if client == nil {
		return 0
	}
	handler := client.events.getURLChanged(WebView(webview))
	if handler == nil {
		return 0
	}
	handler(WebView(webview), utf8FromPtr(url), canGoBack != 0, canGoForward != 0)
	return 0
})

var navigationCallback = syscall.NewCallback(func(webview uintptr, param uintptr, navigationType uintptr, url uintptr) uintptr {
	client := lookupClient(param)
	if client == nil {
		return 1
	}
	handler := client.events.getNavigation(WebView(webview))
	if handler == nil {
		return 1
	}
	if handler(WebView(webview), int32(navigationType), utf8FromPtr(url)) {
		return 1
	}
	return 0
})

var documentReadyCallback = syscall.NewCallback(func(webview uintptr, param uintptr, frameID uintptr) uintptr {
	client := lookupClient(param)
	if client == nil {
		return 0
	}
	handler := client.events.getDocumentReady(WebView(webview))
	if handler == nil {
		return 0
	}
	handler(WebView(webview), WebFrameHandle(frameID))
	return 0
})

var loadURLBeginCallback = syscall.NewCallback(func(webview uintptr, param uintptr, url uintptr, job uintptr) uintptr {
	client := lookupClient(param)
	if client == nil {
		return 0
	}
	handler := client.events.getLoadURLBegin(WebView(webview))
	if handler == nil {
		return 0
	}
	if handler(WebView(webview), utf8FromPtr(url), job) {
		return 1
	}
	return 0
})

var loadURLEndCallback = syscall.NewCallback(func(webview uintptr, param uintptr, url uintptr, job uintptr, buf uintptr, length uintptr) uintptr {
	client := lookupClient(param)
	if client == nil {
		return 0
	}
	handler := client.events.getLoadURLEnd(WebView(webview))
	if handler == nil {
		return 0
	}
	handler(WebView(webview), utf8FromPtr(url), job, buf, int32(length))
	return 0
})

var jsQueryCallback = syscall.NewCallback(func(webview uintptr, param uintptr, execState uintptr, queryID uintptr, customMsg uintptr, request uintptr) uintptr {
	client := lookupClient(param)
	if client == nil {
		return 0
	}
	handler := client.events.getJsQuery(WebView(webview))
	if handler == nil {
		return 0
	}
	handler(JsQuery{
		View:      WebView(webview),
		ExecState: execState,
		QueryID:   int64(queryID),
		CustomMsg: int32(customMsg),
		Request:   utf8FromPtr(request),
		client:    client,
	})
	return 0
})

var createViewCallback = syscall.NewCallback(func(webview uintptr, param uintptr, navigationType uintptr, url uintptr, windowFeatures uintptr) uintptr {
	client := lookupClient(param)
	if client == nil {
		return 0
	}
	handler := client.events.getCreateView(WebView(webview))
	if handler == nil {
		return 0
	}
	newView := handler(WebView(webview), int32(navigationType), utf8FromPtr(url), windowFeatures)
	return uintptr(newView)
})

var closeWindowCallback = syscall.NewCallback(func(webview uintptr, param uintptr) uintptr {
	client := lookupClient(param)
	if client == nil {
		return 1
	}
	handler := client.events.getCloseWindow(WebView(webview))
	if handler == nil {
		return 1
	}
	if handler(WebView(webview)) {
		return 1
	}
	return 0
})

var destroyWindowCallback = syscall.NewCallback(func(webview uintptr, param uintptr) uintptr {
	client := lookupClient(param)
	if client == nil {
		return 0
	}
	handler := client.events.getDestroyWindow(WebView(webview))
	if handler != nil {
		handler(WebView(webview))
	}
	client.events.removeView(WebView(webview))
	return 0
})

var alertBoxCallback = syscall.NewCallback(func(webview uintptr, param uintptr, msg uintptr) uintptr {
	client := lookupClient(param)
	if client == nil {
		return 0
	}
	handler := client.events.getAlertBox(WebView(webview))
	if handler != nil {
		handler(WebView(webview), utf8FromPtr(msg))
	}
	return 0
})

var confirmBoxCallback = syscall.NewCallback(func(webview uintptr, param uintptr, msg uintptr) uintptr {
	client := lookupClient(param)
	if client == nil {
		return 0
	}
	handler := client.events.getConfirmBox(WebView(webview))
	if handler == nil {
		return 0
	}
	if handler(WebView(webview), utf8FromPtr(msg)) {
		return 1
	}
	return 0
})

var promptResultStore sync.Map
var promptResultSeed atomic.Uintptr

var promptBoxCallback = syscall.NewCallback(func(webview uintptr, param uintptr, msg uintptr, defaultResult uintptr) uintptr {
	client := lookupClient(param)
	if client == nil {
		return 0
	}
	handler := client.events.getPromptBox(WebView(webview))
	if handler == nil {
		return 0
	}
	result := handler(WebView(webview), utf8FromPtr(msg), utf8FromPtr(defaultResult))
	b := toCString(result)
	id := promptResultSeed.Add(1)
	promptResultStore.Store(id, b)
	return uintptr(unsafe.Pointer(&b[0]))
})

func utf8FromPtr(ptr uintptr) string {
	if ptr == 0 {
		return ""
	}
	bytes := make([]byte, 0, 64)
	for i := uintptr(0); ; i++ {
		b := *(*byte)(unsafe.Pointer(ptr + i))
		if b == 0 {
			break
		}
		bytes = append(bytes, b)
	}
	return string(bytes)
}

func (c *Client) OnTitleChanged(view WebView, handler TitleChangedHandler) error {
	if err := requireProc(c.dll.mbOnTitleChanged, "mbOnTitleChanged"); err != nil {
		return err
	}
	c.events.setTitleChanged(view, handler)
	c.dll.mbOnTitleChanged.Call(uintptr(view), titleChangedCallback, c.id)
	return nil
}

func (c *Client) OnURLChanged(view WebView, handler URLChangedHandler) error {
	if err := requireProc(c.dll.mbOnURLChanged, "mbOnURLChanged"); err != nil {
		return err
	}
	c.events.setURLChanged(view, handler)
	c.dll.mbOnURLChanged.Call(uintptr(view), urlChangedCallback, c.id)
	return nil
}

func (c *Client) OnNavigation(view WebView, handler NavigationHandler) error {
	if err := requireProc(c.dll.mbOnNavigation, "mbOnNavigation"); err != nil {
		return err
	}
	c.events.setNavigation(view, handler)
	c.dll.mbOnNavigation.Call(uintptr(view), navigationCallback, c.id)
	return nil
}

func (c *Client) OnDocumentReady(view WebView, handler DocumentReadyHandler) error {
	if err := requireProc(c.dll.mbOnDocumentReady, "mbOnDocumentReady"); err != nil {
		return err
	}
	c.events.setDocumentReady(view, handler)
	c.dll.mbOnDocumentReady.Call(uintptr(view), documentReadyCallback, c.id)
	return nil
}

func (c *Client) OnLoadURLBegin(view WebView, handler LoadURLBeginHandler) error {
	if err := requireProc(c.dll.mbOnLoadUrlBegin, "mbOnLoadUrlBegin"); err != nil {
		return err
	}
	c.events.setLoadURLBegin(view, handler)
	c.dll.mbOnLoadUrlBegin.Call(uintptr(view), loadURLBeginCallback, c.id)
	return nil
}

func (c *Client) OnLoadURLEnd(view WebView, handler LoadURLEndHandler) error {
	if err := requireProc(c.dll.mbOnLoadUrlEnd, "mbOnLoadUrlEnd"); err != nil {
		return err
	}
	c.events.setLoadURLEnd(view, handler)
	c.dll.mbOnLoadUrlEnd.Call(uintptr(view), loadURLEndCallback, c.id)
	return nil
}

func (c *Client) OnJsQuery(view WebView, handler JsQueryHandler) error {
	if err := requireProc(c.dll.mbOnJsQuery, "mbOnJsQuery"); err != nil {
		return err
	}
	c.events.setJsQuery(view, handler)
	c.dll.mbOnJsQuery.Call(uintptr(view), jsQueryCallback, c.id)
	return nil
}

func (c *Client) OnCreateView(view WebView, handler CreateViewHandler) error {
	if err := requireProc(c.dll.mbOnCreateView, "mbOnCreateView"); err != nil {
		return err
	}
	c.events.setCreateView(view, handler)
	c.dll.mbOnCreateView.Call(uintptr(view), createViewCallback, c.id)
	return nil
}

func (c *Client) OnClose(view WebView, handler CloseHandler) error {
	if err := requireProc(c.dll.mbOnClose, "mbOnClose"); err != nil {
		return err
	}
	c.events.setCloseWindow(view, handler)
	c.dll.mbOnClose.Call(uintptr(view), closeWindowCallback, c.id)
	return nil
}

func (c *Client) OnDestroy(view WebView, handler DestroyHandler) error {
	if err := requireProc(c.dll.mbOnDestroy, "mbOnDestroy"); err != nil {
		return err
	}
	c.events.setDestroyWindow(view, handler)
	c.dll.mbOnDestroy.Call(uintptr(view), destroyWindowCallback, c.id)
	return nil
}

func (c *Client) OnAlertBox(view WebView, handler AlertBoxHandler) error {
	if err := requireProc(c.dll.mbOnAlertBox, "mbOnAlertBox"); err != nil {
		return err
	}
	c.events.setAlertBox(view, handler)
	c.dll.mbOnAlertBox.Call(uintptr(view), alertBoxCallback, c.id)
	return nil
}

func (c *Client) OnConfirmBox(view WebView, handler ConfirmBoxHandler) error {
	if err := requireProc(c.dll.mbOnConfirmBox, "mbOnConfirmBox"); err != nil {
		return err
	}
	c.events.setConfirmBox(view, handler)
	c.dll.mbOnConfirmBox.Call(uintptr(view), confirmBoxCallback, c.id)
	return nil
}

func (c *Client) OnPromptBox(view WebView, handler PromptBoxHandler) error {
	if err := requireProc(c.dll.mbOnPromptBox, "mbOnPromptBox"); err != nil {
		return err
	}
	c.events.setPromptBox(view, handler)
	c.dll.mbOnPromptBox.Call(uintptr(view), promptBoxCallback, c.id)
	return nil
}

func (c *Client) ResponseQuery(view WebView, queryID int64, customMsg int32, response string) error {
	if err := requireProc(c.dll.mbResponseQuery, "mbResponseQuery"); err != nil {
		return err
	}
	b := toCString(response)
	c.dll.mbResponseQuery.Call(
		uintptr(view),
		uintptr(queryID),
		uintptr(customMsg),
		uintptr(unsafe.Pointer(&b[0])),
	)
	return nil
}

var postUITaskSeed atomic.Uintptr
var postUITasks sync.Map

var postToUIThreadCallback = syscall.NewCallback(func(webview uintptr, param uintptr) uintptr {
	if v, ok := postUITasks.LoadAndDelete(param); ok {
		if fn, ok := v.(func()); ok {
			fn()
		}
	}
	return 0
})

func (c *Client) PostToUIThread(fn func()) error {
	if err := requireProc(c.dll.mbPostToUiThread, "mbPostToUiThread"); err != nil {
		return err
	}
	id := postUITaskSeed.Add(1)
	postUITasks.Store(id, fn)
	c.dll.mbPostToUiThread.Call(postToUIThreadCallback, id)
	return nil
}
