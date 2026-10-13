//go:build windows

package mbvip

import "unsafe"

func (c *Client) RunJS(view WebView, frameID WebFrameHandle, script string, inClosure bool) error {
	if err := requireProc(c.dll.mbRunJS, "mbRunJS"); err != nil {
		return err
	}
	s := toCString(script)
	c.dll.mbRunJS.Call(
		uintptr(view),
		uintptr(c.resolveFrame(view, frameID)),
		uintptr(unsafe.Pointer(&s[0])),
		boolToUintptr(inClosure),
		0,
		0,
		0,
	)
	return nil
}

func (c *Client) RunJSMainFrame(view WebView, script string) error {
	return c.RunJS(view, 0, script, false)
}

func (c *Client) resolveFrame(view WebView, frameID WebFrameHandle) WebFrameHandle {
	if frameID != 0 {
		return frameID
	}
	if c.dll.mbWebFrameGetMainFrame.Find() != nil {
		return 0
	}
	f, _, _ := c.dll.mbWebFrameGetMainFrame.Call(uintptr(view))
	return WebFrameHandle(f)
}
