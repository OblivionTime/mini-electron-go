//go:build windows

package mbvip

import (
	"unsafe"
)

func (c *Client) LoadHTMLWithBaseURL(view WebView, html string, baseURL string) error {
	if err := requireProc(c.dll.mbLoadHtmlWithBaseUrl, "mbLoadHtmlWithBaseUrl"); err != nil {
		return err
	}
	htmlText := toCString(html)
	base := toCString(baseURL)
	c.dll.mbLoadHtmlWithBaseUrl.Call(
		uintptr(view),
		uintptr(unsafe.Pointer(&htmlText[0])),
		uintptr(unsafe.Pointer(&base[0])),
	)
	return nil
}

func (c *Client) StopLoading(view WebView) error {
	if err := requireProc(c.dll.mbStopLoading, "mbStopLoading"); err != nil {
		return err
	}
	c.dll.mbStopLoading.Call(uintptr(view))
	return nil
}

func (c *Client) Reload(view WebView) (bool, error) {
	if err := requireProc(c.dll.mbReload, "mbReload"); err != nil {
		return false, err
	}
	r, _, _ := c.dll.mbReload.Call(uintptr(view))
	return r != 0, nil
}

func (c *Client) GoBack(view WebView) (bool, error) {
	if err := requireProc(c.dll.mbGoBack, "mbGoBack"); err != nil {
		return false, err
	}
	r, _, _ := c.dll.mbGoBack.Call(uintptr(view))
	return r != 0, nil
}

func (c *Client) GoForward(view WebView) (bool, error) {
	if err := requireProc(c.dll.mbGoForward, "mbGoForward"); err != nil {
		return false, err
	}
	r, _, _ := c.dll.mbGoForward.Call(uintptr(view))
	return r != 0, nil
}
