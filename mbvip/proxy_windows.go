//go:build windows

package mbvip

import "unsafe"

func (c *Client) SetProxy(proxy Proxy) error {
	if err := requireProc(c.dll.mbSetProxy, "mbSetProxy"); err != nil {
		return err
	}
	native := toNativeProxy(proxy)
	c.dll.mbSetProxy.Call(uintptr(unsafe.Pointer(&native)))
	return nil
}

func (c *Client) SetViewProxy(view WebView, proxy Proxy) error {
	if err := requireProc(c.dll.mbSetViewProxy, "mbSetViewProxy"); err != nil {
		return err
	}
	native := toNativeProxy(proxy)
	c.dll.mbSetViewProxy.Call(uintptr(view), uintptr(unsafe.Pointer(&native)))
	return nil
}

func toNativeProxy(proxy Proxy) mbProxy {
	native := mbProxy{
		Type: int32(proxy.Type),
		Port: proxy.Port,
	}
	copy(native.Hostname[:], []byte(proxy.Hostname))
	copy(native.Username[:], []byte(proxy.Username))
	copy(native.Password[:], []byte(proxy.Password))
	return native
}
