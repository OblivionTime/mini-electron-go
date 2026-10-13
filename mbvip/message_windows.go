//go:build windows

package mbvip

func (c *Client) SetFocus(view WebView) error {
	if err := requireProc(c.dll.mbSetFocus, "mbSetFocus"); err != nil {
		return err
	}
	c.dll.mbSetFocus.Call(uintptr(view))
	return nil
}

func (c *Client) KillFocus(view WebView) error {
	if err := requireProc(c.dll.mbKillFocus, "mbKillFocus"); err != nil {
		return err
	}
	c.dll.mbKillFocus.Call(uintptr(view))
	return nil
}

func (c *Client) FireMouseEvent(view WebView, message uint32, x int32, y int32, flags uint32) (bool, error) {
	if err := requireProc(c.dll.mbFireMouseEvent, "mbFireMouseEvent"); err != nil {
		return false, err
	}
	r, _, _ := c.dll.mbFireMouseEvent.Call(
		uintptr(view),
		uintptr(message),
		uintptr(x),
		uintptr(y),
		uintptr(flags),
	)
	return r != 0, nil
}

func (c *Client) FireMouseWheelEvent(view WebView, x int32, y int32, delta int32, flags uint32) (bool, error) {
	if err := requireProc(c.dll.mbFireMouseWheelEvent, "mbFireMouseWheelEvent"); err != nil {
		return false, err
	}
	r, _, _ := c.dll.mbFireMouseWheelEvent.Call(
		uintptr(view),
		uintptr(x),
		uintptr(y),
		uintptr(delta),
		uintptr(flags),
	)
	return r != 0, nil
}

func (c *Client) FireKeyUpEvent(view WebView, virtualKeyCode uint32, flags uint32, systemKey bool) (bool, error) {
	if err := requireProc(c.dll.mbFireKeyUpEvent, "mbFireKeyUpEvent"); err != nil {
		return false, err
	}
	r, _, _ := c.dll.mbFireKeyUpEvent.Call(
		uintptr(view),
		uintptr(virtualKeyCode),
		uintptr(flags),
		boolToUintptr(systemKey),
	)
	return r != 0, nil
}

func (c *Client) FireKeyDownEvent(view WebView, virtualKeyCode uint32, flags uint32, systemKey bool) (bool, error) {
	if err := requireProc(c.dll.mbFireKeyDownEvent, "mbFireKeyDownEvent"); err != nil {
		return false, err
	}
	r, _, _ := c.dll.mbFireKeyDownEvent.Call(
		uintptr(view),
		uintptr(virtualKeyCode),
		uintptr(flags),
		boolToUintptr(systemKey),
	)
	return r != 0, nil
}

func (c *Client) FireKeyPressEvent(view WebView, charCode uint32, flags uint32, systemKey bool) (bool, error) {
	if err := requireProc(c.dll.mbFireKeyPressEvent, "mbFireKeyPressEvent"); err != nil {
		return false, err
	}
	r, _, _ := c.dll.mbFireKeyPressEvent.Call(
		uintptr(view),
		uintptr(charCode),
		uintptr(flags),
		boolToUintptr(systemKey),
	)
	return r != 0, nil
}

func (c *Client) FireWindowsMessage(view WebView, hwnd uintptr, message uint32, wParam uintptr, lParam uintptr, resultPtr uintptr) (bool, error) {
	if err := requireProc(c.dll.mbFireWindowsMessage, "mbFireWindowsMessage"); err != nil {
		return false, err
	}
	r, _, _ := c.dll.mbFireWindowsMessage.Call(
		uintptr(view),
		hwnd,
		uintptr(message),
		wParam,
		lParam,
		resultPtr,
	)
	return r != 0, nil
}
