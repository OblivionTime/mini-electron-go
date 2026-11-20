//go:build windows

package mbvip

import (
	"fmt"
	"syscall"
	"unsafe"
)

func (c *Client) CreateWebView() (WebView, error) {
	if err := requireProc(c.dll.mbCreateWebView, "mbCreateWebView"); err != nil {
		return 0, err
	}
	h, _, _ := c.dll.mbCreateWebView.Call()
	if h == 0 {
		return 0, fmt.Errorf("mbCreateWebView returned null handle")
	}
	return WebView(h), nil
}

func (c *Client) DestroyWebView(view WebView) error {
	if err := requireProc(c.dll.mbDestroyWebView, "mbDestroyWebView"); err != nil {
		return err
	}
	c.dll.mbDestroyWebView.Call(uintptr(view))
	return nil
}

func (c *Client) DestroyWebWindow(view WebView) error {
	if err := requireProc(c.dll.mbDestroyWebWindow, "mbDestroyWebWindow"); err != nil {
		return err
	}
	c.dll.mbDestroyWebWindow.Call(uintptr(view))
	return nil
}

func (c *Client) Resize(view WebView, width int, height int) error {
	if width <= 0 || height <= 0 {
		return fmt.Errorf("invalid window size: %dx%d", width, height)
	}
	if err := requireProc(c.dll.mbResize, "mbResize"); err != nil {
		return err
	}
	c.dll.mbResize.Call(uintptr(view), uintptr(width), uintptr(height))
	return nil
}

func (c *Client) HostHWND(view WebView) (uintptr, error) {
	if err := requireProc(c.dll.mbGetHostHWND, "mbGetHostHWND"); err != nil {
		return 0, err
	}
	hwnd, _, _ := c.dll.mbGetHostHWND.Call(uintptr(view))
	return hwnd, nil
}

const swMaximize = 3

// MaximizeWindow 让承载该 view 的原生窗口最大化（含任务栏/工作区处理）。
func (c *Client) MaximizeWindow(view WebView) error {
	hwnd, err := c.HostHWND(view)
	if err != nil {
		return err
	}
	if hwnd == 0 {
		return syscall.EINVAL
	}
	user32 := syscall.NewLazyDLL("user32.dll")
	showWindow := user32.NewProc("ShowWindow")
	if err := showWindow.Find(); err != nil {
		return err
	}
	showWindow.Call(hwnd, swMaximize)
	return nil
}

func (c *Client) SetWindowTitle(view WebView, title string) error {
	if err := requireProc(c.dll.mbSetWindowTitle, "mbSetWindowTitle"); err != nil {
		return err
	}
	b := toCString(title)
	c.dll.mbSetWindowTitle.Call(uintptr(view), uintptr(unsafe.Pointer(&b[0])))
	return nil
}

const (
	wmSetIcon      = 0x0080
	iconSmall      = 0
	iconBig        = 1
	imageIcon      = 1
	lrDefaultSize  = 0x0040
	lrShared       = 0x8000
	lrLoadFromFile = 0x0010
)

// SetWindowIconFromFile loads an .ico from disk and applies it to the window's
// title bar and taskbar button.
func SetWindowIconFromFile(hwnd uintptr, icoPath string) error {
	if hwnd == 0 {
		return syscall.EINVAL
	}
	user32 := syscall.NewLazyDLL("user32.dll")
	loadImage := user32.NewProc("LoadImageW")
	sendMessage := user32.NewProc("SendMessageW")

	p := toUTF16(icoPath)
	hIcon, _, callErr := loadImage.Call(
		0,
		uintptr(unsafe.Pointer(&p[0])),
		imageIcon,
		0,
		0,
		lrLoadFromFile|lrDefaultSize,
	)
	if hIcon == 0 {
		if callErr != nil && callErr != syscall.Errno(0) {
			return callErr
		}
		return syscall.EINVAL
	}

	sendMessage.Call(hwnd, wmSetIcon, iconBig, hIcon)
	sendMessage.Call(hwnd, wmSetIcon, iconSmall, hIcon)
	return nil
}

func SetWindowIconFromResource(hwnd uintptr) error {
	user32 := syscall.NewLazyDLL("user32.dll")
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	loadImage := user32.NewProc("LoadImageW")
	sendMessage := user32.NewProc("SendMessageW")
	getModuleHandle := kernel32.NewProc("GetModuleHandleW")

	hInst, _, _ := getModuleHandle.Call(0)
	if hInst == 0 {
		return syscall.EINVAL
	}

	hIcon, _, callErr := loadImage.Call(
		hInst,
		1,
		imageIcon,
		0,
		0,
		lrDefaultSize|lrShared,
	)
	if hIcon == 0 {
		if callErr != nil && callErr != syscall.Errno(0) {
			return callErr
		}
		return syscall.EINVAL
	}

	sendMessage.Call(hwnd, wmSetIcon, iconBig, hIcon)
	sendMessage.Call(hwnd, wmSetIcon, iconSmall, hIcon)
	return nil
}
