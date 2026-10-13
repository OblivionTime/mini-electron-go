//go:build windows

package mbvip

import "unsafe"

func (c *Client) SetCookie(view WebView, url string, cookie string) error {
	if err := requireProc(c.dll.mbSetCookie, "mbSetCookie"); err != nil {
		return err
	}
	urlText := toCString(url)
	cookieText := toCString(cookie)
	c.dll.mbSetCookie.Call(
		uintptr(view),
		uintptr(unsafe.Pointer(&urlText[0])),
		uintptr(unsafe.Pointer(&cookieText[0])),
	)
	return nil
}

func (c *Client) ClearCookie(view WebView) error {
	if err := requireProc(c.dll.mbClearCookie, "mbClearCookie"); err != nil {
		return err
	}
	c.dll.mbClearCookie.Call(uintptr(view))
	return nil
}

func (c *Client) SetCookieEnabled(view WebView, enable bool) error {
	if err := requireProc(c.dll.mbSetCookieEnabled, "mbSetCookieEnabled"); err != nil {
		return err
	}
	c.dll.mbSetCookieEnabled.Call(uintptr(view), boolToUintptr(enable))
	return nil
}

func (c *Client) SetCookieJarPath(view WebView, dir string) error {
	if err := requireProc(c.dll.mbSetCookieJarPath, "mbSetCookieJarPath"); err != nil {
		return err
	}
	p := toUTF16(dir)
	c.dll.mbSetCookieJarPath.Call(uintptr(view), uintptr(unsafe.Pointer(&p[0])))
	return nil
}

func (c *Client) SetCookieJarFullPath(view WebView, filePath string) error {
	if err := requireProc(c.dll.mbSetCookieJarFullPath, "mbSetCookieJarFullPath"); err != nil {
		return err
	}
	p := toUTF16(filePath)
	c.dll.mbSetCookieJarFullPath.Call(uintptr(view), uintptr(unsafe.Pointer(&p[0])))
	return nil
}

func (c *Client) SetLocalStorageFullPath(view WebView, dir string) error {
	if err := requireProc(c.dll.mbSetLocalStorageFullPath, "mbSetLocalStorageFullPath"); err != nil {
		return err
	}
	p := toUTF16(dir)
	c.dll.mbSetLocalStorageFullPath.Call(uintptr(view), uintptr(unsafe.Pointer(&p[0])))
	return nil
}

func (c *Client) SetDiskCacheEnabled(view WebView, enable bool) error {
	if err := requireProc(c.dll.mbSetDiskCacheEnabled, "mbSetDiskCacheEnabled"); err != nil {
		return err
	}
	c.dll.mbSetDiskCacheEnabled.Call(uintptr(view), boolToUintptr(enable))
	return nil
}

func (c *Client) SetDiskCachePath(view WebView, dir string) error {
	if err := requireProc(c.dll.mbSetDiskCachePath, "mbSetDiskCachePath"); err != nil {
		return err
	}
	p := toUTF16(dir)
	c.dll.mbSetDiskCachePath.Call(uintptr(view), uintptr(unsafe.Pointer(&p[0])))
	return nil
}

func (c *Client) SetDiskCacheLimit(view WebView, bytes uint64) error {
	if err := requireProc(c.dll.mbSetDiskCacheLimit, "mbSetDiskCacheLimit"); err != nil {
		return err
	}
	c.dll.mbSetDiskCacheLimit.Call(uintptr(view), uintptr(bytes))
	return nil
}

func (c *Client) SetDiskCacheLimitDisk(view WebView, bytes uint64) error {
	if err := requireProc(c.dll.mbSetDiskCacheLimitDisk, "mbSetDiskCacheLimitDisk"); err != nil {
		return err
	}
	c.dll.mbSetDiskCacheLimitDisk.Call(uintptr(view), uintptr(bytes))
	return nil
}

func (c *Client) SetDiskCacheLevel(view WebView, level int) error {
	if err := requireProc(c.dll.mbSetDiskCacheLevel, "mbSetDiskCacheLevel"); err != nil {
		return err
	}
	c.dll.mbSetDiskCacheLevel.Call(uintptr(view), uintptr(level))
	return nil
}
