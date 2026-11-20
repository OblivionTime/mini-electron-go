//go:build windows

package mbvip

import "unsafe"

func (c *Client) GetURL(view WebView) (string, error) {
	if err := requireProc(c.dll.mbGetURL, "mbGetUrl"); err != nil {
		return "", err
	}
	p, _, _ := c.dll.mbGetURL.Call(uintptr(view))
	return utf8FromPtr(p), nil
}

func (c *Client) SetHandle(view WebView, hwnd uintptr) error {
	if err := requireProc(c.dll.mbSetHandle, "mbSetHandle"); err != nil {
		return err
	}
	c.dll.mbSetHandle.Call(uintptr(view), hwnd)
	return nil
}

func (c *Client) SetHandleOffset(view WebView, x int32, y int32) error {
	if err := requireProc(c.dll.mbSetHandleOffset, "mbSetHandleOffset"); err != nil {
		return err
	}
	c.dll.mbSetHandleOffset.Call(uintptr(view), uintptr(x), uintptr(y))
	return nil
}

func (c *Client) WebFrameGetMainFrame(view WebView) (WebFrameHandle, error) {
	if err := requireProc(c.dll.mbWebFrameGetMainFrame, "mbWebFrameGetMainFrame"); err != nil {
		return 0, err
	}
	f, _, _ := c.dll.mbWebFrameGetMainFrame.Call(uintptr(view))
	return WebFrameHandle(f), nil
}

func (c *Client) IsMainFrame(view WebView, frameID WebFrameHandle) (bool, error) {
	if err := requireProc(c.dll.mbIsMainFrame, "mbIsMainFrame"); err != nil {
		return false, err
	}
	r, _, _ := c.dll.mbIsMainFrame.Call(uintptr(view), uintptr(frameID))
	return byte(r) != 0, nil
}

func (c *Client) EditorSelectAll(view WebView) error {
	if err := requireProc(c.dll.mbEditorSelectAll, "mbEditorSelectAll"); err != nil {
		return err
	}
	c.dll.mbEditorSelectAll.Call(uintptr(view))
	return nil
}

func (c *Client) EditorUnSelect(view WebView) error {
	if err := requireProc(c.dll.mbEditorUnSelect, "mbEditorUnSelect"); err != nil {
		return err
	}
	c.dll.mbEditorUnSelect.Call(uintptr(view))
	return nil
}

func (c *Client) EditorCopy(view WebView) error {
	if err := requireProc(c.dll.mbEditorCopy, "mbEditorCopy"); err != nil {
		return err
	}
	c.dll.mbEditorCopy.Call(uintptr(view))
	return nil
}

func (c *Client) EditorCut(view WebView) error {
	if err := requireProc(c.dll.mbEditorCut, "mbEditorCut"); err != nil {
		return err
	}
	c.dll.mbEditorCut.Call(uintptr(view))
	return nil
}

func (c *Client) EditorDelete(view WebView) error {
	if err := requireProc(c.dll.mbEditorDelete, "mbEditorDelete"); err != nil {
		return err
	}
	c.dll.mbEditorDelete.Call(uintptr(view))
	return nil
}

func (c *Client) EditorUndo(view WebView) error {
	if err := requireProc(c.dll.mbEditorUndo, "mbEditorUndo"); err != nil {
		return err
	}
	c.dll.mbEditorUndo.Call(uintptr(view))
	return nil
}

func (c *Client) EditorRedo(view WebView) error {
	if err := requireProc(c.dll.mbEditorRedo, "mbEditorRedo"); err != nil {
		return err
	}
	c.dll.mbEditorRedo.Call(uintptr(view))
	return nil
}

func (c *Client) GetCookieW(view WebView, callback uintptr, param uintptr) error {
	if err := requireProc(c.dll.mbGetCookieW, "mbGetCookieW"); err != nil {
		return err
	}
	c.dll.mbGetCookieW.Call(uintptr(view), callback, param)
	return nil
}

func (c *Client) VisitAllCookie(params uintptr, visitor uintptr) error {
	if err := requireProc(c.dll.mbVisitAllCookie, "mbVisitAllCookie"); err != nil {
		return err
	}
	c.dll.mbVisitAllCookie.Call(params, visitor)
	return nil
}

func (c *Client) PerformCookieCommand(command CookieCommand) error {
	if err := requireProc(c.dll.mbPerformCookieCommand, "mbPerformCookieCommand"); err != nil {
		return err
	}
	c.dll.mbPerformCookieCommand.Call(uintptr(command))
	return nil
}

func (c *Client) NetSetHTTPHeaderField(jobPtr uintptr, key string, value string, response bool) error {
	if err := requireProc(c.dll.mbNetSetHTTPHeaderField, "mbNetSetHTTPHeaderField"); err != nil {
		return err
	}
	k := toUTF16(key)
	v := toUTF16(value)
	c.dll.mbNetSetHTTPHeaderField.Call(
		jobPtr,
		uintptr(unsafe.Pointer(&k[0])),
		uintptr(unsafe.Pointer(&v[0])),
		boolToUintptr(response),
	)
	return nil
}

func (c *Client) NetGetRawHTTPHead(jobPtr uintptr) (uintptr, error) {
	if err := requireProc(c.dll.mbNetGetRawHttpHead, "mbNetGetRawHttpHead"); err != nil {
		return 0, err
	}
	p, _, _ := c.dll.mbNetGetRawHttpHead.Call(jobPtr)
	return p, nil
}

func (c *Client) NetSetMIMEType(jobPtr uintptr, mimeType string) error {
	if err := requireProc(c.dll.mbNetSetMIMEType, "mbNetSetMIMEType"); err != nil {
		return err
	}
	m := toCString(mimeType)
	c.dll.mbNetSetMIMEType.Call(jobPtr, uintptr(unsafe.Pointer(&m[0])))
	return nil
}

func (c *Client) NetGetMIMEType(jobPtr uintptr) (string, error) {
	if err := requireProc(c.dll.mbNetGetMIMEType, "mbNetGetMIMEType"); err != nil {
		return "", err
	}
	p, _, _ := c.dll.mbNetGetMIMEType.Call(jobPtr, 0)
	return utf8FromPtr(p), nil
}

func (c *Client) NetSetData(jobPtr uintptr, data []byte) error {
	if err := requireProc(c.dll.mbNetSetData, "mbNetSetData"); err != nil {
		return err
	}
	if len(data) == 0 {
		c.dll.mbNetSetData.Call(jobPtr, 0, 0)
		return nil
	}
	c.dll.mbNetSetData.Call(jobPtr, uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)))
	return nil
}

func (c *Client) NetHookRequest(jobPtr uintptr) error {
	if err := requireProc(c.dll.mbNetHookRequest, "mbNetHookRequest"); err != nil {
		return err
	}
	c.dll.mbNetHookRequest.Call(jobPtr)
	return nil
}

func (c *Client) NetHoldJobToAsynCommit(jobPtr uintptr) (bool, error) {
	if err := requireProc(c.dll.mbNetHoldJobToAsynCommit, "mbNetHoldJobToAsynCommit"); err != nil {
		return false, err
	}
	r, _, _ := c.dll.mbNetHoldJobToAsynCommit.Call(jobPtr)
	return r != 0, nil
}

func (c *Client) NetContinueJob(jobPtr uintptr) error {
	if err := requireProc(c.dll.mbNetContinueJob, "mbNetContinueJob"); err != nil {
		return err
	}
	c.dll.mbNetContinueJob.Call(jobPtr)
	return nil
}

func (c *Client) NetGetRequestMethod(jobPtr uintptr) (RequestType, error) {
	if err := requireProc(c.dll.mbNetGetRequestMethod, "mbNetGetRequestMethod"); err != nil {
		return 0, err
	}
	r, _, _ := c.dll.mbNetGetRequestMethod.Call(jobPtr)
	return RequestType(r), nil
}

func (c *Client) NetGetPostBody(jobPtr uintptr) (uintptr, error) {
	if err := requireProc(c.dll.mbNetGetPostBody, "mbNetGetPostBody"); err != nil {
		return 0, err
	}
	p, _, _ := c.dll.mbNetGetPostBody.Call(jobPtr)
	return p, nil
}

func (c *Client) ToDouble(execState uintptr, value uintptr) (float64, error) {
	if err := requireProc(c.dll.mbToDouble, "mbToDouble"); err != nil {
		return 0, err
	}
	r, _, _ := c.dll.mbToDouble.Call(execState, value)
	return uintptrToFloat64(r), nil
}

func (c *Client) ToString(execState uintptr, value uintptr) (string, error) {
	if err := requireProc(c.dll.mbToString, "mbToString"); err != nil {
		return "", err
	}
	p, _, _ := c.dll.mbToString.Call(execState, value)
	return utf8FromPtr(p), nil
}

func (c *Client) GetString(s MbString) (string, error) {
	if err := requireProc(c.dll.mbGetString, "mbGetString"); err != nil {
		return "", err
	}
	p, _, _ := c.dll.mbGetString.Call(uintptr(s))
	return utf8FromPtr(p), nil
}

func (c *Client) SetString(s MbString, text string) error {
	if err := requireProc(c.dll.mbSetString, "mbSetString"); err != nil {
		return err
	}
	b := toCString(text)
	c.dll.mbSetString.Call(uintptr(s), uintptr(unsafe.Pointer(&b[0])), uintptr(len(text)))
	return nil
}

func (c *Client) DeleteString(s MbString) error {
	if err := requireProc(c.dll.mbDeleteString, "mbDeleteString"); err != nil {
		return err
	}
	c.dll.mbDeleteString.Call(uintptr(s))
	return nil
}
