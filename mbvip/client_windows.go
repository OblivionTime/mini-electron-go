//go:build windows

package mbvip

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"
)

func (c *Client) Init() {
	runtime.LockOSThread()
	c.dll.mbInit.Call(0)
}

func (c *Client) uninit() {
	defer runtime.UnlockOSThread()
	p := c.dll.dll.NewProc("mbUninit")
	if p.Find() == nil {
		p.Call()
	}
}

func (c *Client) exitMessageLoop() {
	p := c.dll.dll.NewProc("mbExitMessageLoop")
	if c.dll.mbRunMessageLoop.Find() == nil && p.Find() == nil {
		p.Call()
		return
	}
	syscall.NewLazyDLL("user32.dll").NewProc("PostQuitMessage").Call(0)
}

func (c *Client) CreateWebWindow(opts WindowOptions) (WebView, error) {
	if opts.Width <= 0 || opts.Height <= 0 {
		return 0, fmt.Errorf("invalid window size: %dx%d", opts.Width, opts.Height)
	}

	h, _, _ := c.dll.mbCreateWebWindow.Call(
		uintptr(opts.Type),
		opts.Parent,
		uintptr(opts.X),
		uintptr(opts.Y),
		uintptr(opts.Width),
		uintptr(opts.Height),
	)
	if h == 0 {
		return 0, fmt.Errorf("mbCreateWebWindow returned null handle")
	}
	return WebView(h), nil
}

func (c *Client) MoveToCenter(view WebView) {
	c.dll.mbMoveToCenter.Call(uintptr(view))
}

func (c *Client) ShowWindow(view WebView, show bool) {
	c.dll.mbShowWindow.Call(uintptr(view), boolToUintptr(show))
}

func (c *Client) LoadURL(view WebView, url string) {
	b := toCString(url)
	c.dll.mbLoadURL.Call(uintptr(view), uintptr(unsafe.Pointer(&b[0])))
}

func (c *Client) SetNavigationToNewWindowEnable(view WebView, enable bool) error {
	if c.dll.mbSetNavigationToNewWindowEnable.Find() != nil {
		return ProcNotFoundError{Name: "mbSetNavigationToNewWindowEnable"}
	}
	c.dll.mbSetNavigationToNewWindowEnable.Call(uintptr(view), boolToUintptr(enable))
	return nil
}

func (c *Client) RunMessageLoop() error {
	if c.dll.mbRunMessageLoop.Find() == nil {
		c.dll.mbRunMessageLoop.Call()
		return nil
	}

	return runWin32MessageLoop()
}

type win32Msg struct {
	hwnd     uintptr
	message  uint32
	wParam   uintptr
	lParam   uintptr
	time     uint32
	pt       win32Point
	lPrivate uint32
}

type win32Point struct {
	x int32
	y int32
}

func runWin32MessageLoop() error {
	user32 := syscall.NewLazyDLL("user32.dll")
	getMessage := user32.NewProc("GetMessageW")
	translate := user32.NewProc("TranslateMessage")
	dispatch := user32.NewProc("DispatchMessageW")

	if err := getMessage.Find(); err != nil {
		return ProcNotFoundError{Name: "GetMessageW"}
	}
	if err := translate.Find(); err != nil {
		return ProcNotFoundError{Name: "TranslateMessage"}
	}
	if err := dispatch.Find(); err != nil {
		return ProcNotFoundError{Name: "DispatchMessageW"}
	}

	var msg win32Msg
	for {
		ret, _, callErr := getMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(ret) == -1 {
			return fmt.Errorf("GetMessageW failed: %w", callErr)
		}
		if ret == 0 {
			return nil
		}
		translate.Call(uintptr(unsafe.Pointer(&msg)))
		dispatch.Call(uintptr(unsafe.Pointer(&msg)))
	}
}
