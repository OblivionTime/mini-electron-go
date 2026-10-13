//go:build windows

package mbvip

import "unsafe"

func (c *Client) SetCSPCheckEnable(view WebView, enable bool) error {
	if err := requireProc(c.dll.mbSetCspCheckEnable, "mbSetCspCheckEnable"); err != nil {
		return err
	}
	c.dll.mbSetCspCheckEnable.Call(uintptr(view), boolToUintptr(enable))
	return nil
}

func (c *Client) SetHeadlessEnable(view WebView, enable bool) error {
	if err := requireProc(c.dll.mbSetHeadlessEnabled, "mbSetHeadlessEnabled"); err != nil {
		return err
	}
	c.dll.mbSetHeadlessEnabled.Call(uintptr(view), boolToUintptr(enable))
	return nil
}

func (c *Client) SetNpapiPluginsEnable(view WebView, enable bool) error {
	if err := requireProc(c.dll.mbSetNpapiPluginsEnabled, "mbSetNpapiPluginsEnabled"); err != nil {
		return err
	}
	c.dll.mbSetNpapiPluginsEnabled.Call(uintptr(view), boolToUintptr(enable))
	return nil
}

func (c *Client) SetMemoryCacheEnable(view WebView, enable bool) error {
	if err := requireProc(c.dll.mbSetMemoryCacheEnable, "mbSetMemoryCacheEnable"); err != nil {
		return err
	}
	c.dll.mbSetMemoryCacheEnable.Call(uintptr(view), boolToUintptr(enable))
	return nil
}

func (c *Client) SetUserAgent(view WebView, userAgent string) error {
	if err := requireProc(c.dll.mbSetUserAgent, "mbSetUserAgent"); err != nil {
		return err
	}
	b := toCString(userAgent)
	c.dll.mbSetUserAgent.Call(uintptr(view), uintptr(unsafe.Pointer(&b[0])))
	return nil
}

func (c *Client) SetZoomFactor(view WebView, factor float32) error {
	if err := requireProc(c.dll.mbSetZoomFactor, "mbSetZoomFactor"); err != nil {
		return err
	}
	c.dll.mbSetZoomFactor.Call(uintptr(view), float32ToUintptr(factor))
	return nil
}

func (c *Client) GetZoomFactor(view WebView) (float32, error) {
	if err := requireProc(c.dll.mbGetZoomFactor, "mbGetZoomFactor"); err != nil {
		return 0, err
	}
	r, _, _ := c.dll.mbGetZoomFactor.Call(uintptr(view))
	return uintptrToFloat32(r), nil
}
