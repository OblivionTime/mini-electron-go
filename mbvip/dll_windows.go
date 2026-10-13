//go:build windows

package mbvip

import "syscall"

type dllFacade struct {
	dll *syscall.LazyDLL

	mbInit            *syscall.LazyProc
	mbRunMessageLoop  *syscall.LazyProc
	mbSetDebugConfig  *syscall.LazyProc
	wkeSetDebugConfig *syscall.LazyProc

	mbCreateWebView    *syscall.LazyProc
	mbDestroyWebView   *syscall.LazyProc
	mbDestroyWebWindow *syscall.LazyProc
	mbCreateWebWindow  *syscall.LazyProc
	mbMoveToCenter     *syscall.LazyProc
	mbShowWindow       *syscall.LazyProc
	mbSetWindowTitle   *syscall.LazyProc
	mbResize           *syscall.LazyProc
	mbGetHostHWND      *syscall.LazyProc

	mbLoadURL              *syscall.LazyProc
	mbGetURL               *syscall.LazyProc
	mbLoadHtmlWithBaseUrl  *syscall.LazyProc
	mbStopLoading          *syscall.LazyProc
	mbReload               *syscall.LazyProc
	mbGoBack               *syscall.LazyProc
	mbGoForward            *syscall.LazyProc
	mbSetHandle            *syscall.LazyProc
	mbSetHandleOffset      *syscall.LazyProc
	mbWebFrameGetMainFrame *syscall.LazyProc
	mbIsMainFrame          *syscall.LazyProc

	mbSetNavigationToNewWindowEnable *syscall.LazyProc
	mbSetCspCheckEnable              *syscall.LazyProc
	mbSetHeadlessEnabled             *syscall.LazyProc
	mbSetNpapiPluginsEnabled         *syscall.LazyProc
	mbSetMemoryCacheEnable           *syscall.LazyProc
	mbSetProxy                       *syscall.LazyProc
	mbSetViewProxy                   *syscall.LazyProc
	mbSetUserAgent                   *syscall.LazyProc
	mbSetZoomFactor                  *syscall.LazyProc
	mbGetZoomFactor                  *syscall.LazyProc
	mbSetFocus                       *syscall.LazyProc
	mbKillFocus                      *syscall.LazyProc
	mbFireMouseEvent                 *syscall.LazyProc
	mbFireMouseWheelEvent            *syscall.LazyProc
	mbFireKeyUpEvent                 *syscall.LazyProc
	mbFireKeyDownEvent               *syscall.LazyProc
	mbFireKeyPressEvent              *syscall.LazyProc
	mbFireWindowsMessage             *syscall.LazyProc

	mbSetCookie               *syscall.LazyProc
	mbGetCookieW              *syscall.LazyProc
	mbVisitAllCookie          *syscall.LazyProc
	mbPerformCookieCommand    *syscall.LazyProc
	mbClearCookie             *syscall.LazyProc
	mbSetCookieEnabled        *syscall.LazyProc
	mbSetCookieJarPath        *syscall.LazyProc
	mbSetCookieJarFullPath    *syscall.LazyProc
	mbSetLocalStorageFullPath *syscall.LazyProc
	mbSetDiskCacheEnabled     *syscall.LazyProc

	mbEditorSelectAll *syscall.LazyProc
	mbEditorUnSelect  *syscall.LazyProc
	mbEditorCopy      *syscall.LazyProc
	mbEditorCut       *syscall.LazyProc
	mbEditorDelete    *syscall.LazyProc
	mbEditorUndo      *syscall.LazyProc
	mbEditorRedo      *syscall.LazyProc

	mbRunJS        *syscall.LazyProc
	mbToDouble     *syscall.LazyProc
	mbToString     *syscall.LazyProc
	mbGetString    *syscall.LazyProc
	mbGetStringW   *syscall.LazyProc
	mbSetString    *syscall.LazyProc
	mbDeleteString *syscall.LazyProc

	mbPostToUiThread *syscall.LazyProc

	mbSetDiskCachePath      *syscall.LazyProc
	mbSetDiskCacheLimit     *syscall.LazyProc
	mbSetDiskCacheLimitDisk *syscall.LazyProc
	mbSetDiskCacheLevel     *syscall.LazyProc

	mbOnTitleChanged  *syscall.LazyProc
	mbOnURLChanged    *syscall.LazyProc
	mbOnNavigation    *syscall.LazyProc
	mbOnDocumentReady *syscall.LazyProc
	mbOnLoadUrlBegin  *syscall.LazyProc
	mbOnLoadUrlEnd    *syscall.LazyProc
	mbOnJsQuery       *syscall.LazyProc
	mbResponseQuery   *syscall.LazyProc
	mbOnCreateView    *syscall.LazyProc
	mbOnClose         *syscall.LazyProc
	mbOnDestroy       *syscall.LazyProc
	mbOnAlertBox      *syscall.LazyProc
	mbOnConfirmBox    *syscall.LazyProc
	mbOnPromptBox     *syscall.LazyProc

	mbNetSetHTTPHeaderField  *syscall.LazyProc
	mbNetGetRawHttpHead      *syscall.LazyProc
	mbNetSetMIMEType         *syscall.LazyProc
	mbNetGetMIMEType         *syscall.LazyProc
	mbNetSetData             *syscall.LazyProc
	mbNetHookRequest         *syscall.LazyProc
	mbNetHoldJobToAsynCommit *syscall.LazyProc
	mbNetContinueJob         *syscall.LazyProc
	mbNetGetRequestMethod    *syscall.LazyProc
	mbNetGetPostBody         *syscall.LazyProc
}

func loadDLL(path string) (*dllFacade, error) {
	dll := syscall.NewLazyDLL(path)
	facade := &dllFacade{dll: dll}

	facade.mbInit = dll.NewProc("mbInit")
	facade.mbCreateWebWindow = dll.NewProc("mbCreateWebWindow")
	facade.mbCreateWebView = dll.NewProc("mbCreateWebView")
	facade.mbDestroyWebView = dll.NewProc("mbDestroyWebView")
	facade.mbDestroyWebWindow = dll.NewProc("mbDestroyWebView")
	facade.mbMoveToCenter = dll.NewProc("mbMoveToCenter")
	facade.mbShowWindow = dll.NewProc("mbShowWindow")
	facade.mbSetWindowTitle = dll.NewProc("mbSetWindowTitle")
	facade.mbResize = dll.NewProc("mbResize")
	facade.mbGetHostHWND = dll.NewProc("mbGetHostHWND")
	facade.mbLoadURL = dll.NewProc("mbLoadURL")
	facade.mbGetURL = dll.NewProc("mbGetURL")
	facade.mbLoadHtmlWithBaseUrl = dll.NewProc("mbLoadHtmlWithBaseUrl")
	facade.mbStopLoading = dll.NewProc("mbStopLoading")
	facade.mbReload = dll.NewProc("mbReload")
	facade.mbGoBack = dll.NewProc("mbGoBack")
	facade.mbGoForward = dll.NewProc("mbGoForward")
	facade.mbSetHandle = dll.NewProc("mbSetHandle")
	facade.mbSetHandleOffset = dll.NewProc("mbSetHandleOffset")
	facade.mbWebFrameGetMainFrame = dll.NewProc("mbWebFrameGetMainFrame")
	facade.mbIsMainFrame = dll.NewProc("mbIsMainFrame")

	facade.mbRunMessageLoop = dll.NewProc("mbRunMessageLoop")
	facade.mbSetDebugConfig = dll.NewProc("mbSetDebugConfig")
	facade.wkeSetDebugConfig = dll.NewProc("wkeSetDebugConfig")
	facade.mbSetCspCheckEnable = dll.NewProc("mbSetCspCheckEnable")
	facade.mbSetHeadlessEnabled = dll.NewProc("mbSetHeadlessEnabled")
	facade.mbSetNpapiPluginsEnabled = dll.NewProc("mbSetNpapiPluginsEnabled")
	facade.mbSetMemoryCacheEnable = dll.NewProc("mbSetMemoryCacheEnable")
	facade.mbSetProxy = dll.NewProc("mbSetProxy")
	facade.mbSetViewProxy = dll.NewProc("mbSetViewProxy")
	facade.mbSetNavigationToNewWindowEnable = dll.NewProc("mbSetNavigationToNewWindowEnable")
	facade.mbSetUserAgent = dll.NewProc("mbSetUserAgent")
	facade.mbSetZoomFactor = dll.NewProc("mbSetZoomFactor")
	facade.mbGetZoomFactor = dll.NewProc("mbGetZoomFactor")
	facade.mbSetFocus = dll.NewProc("mbSetFocus")
	facade.mbKillFocus = dll.NewProc("mbKillFocus")
	facade.mbFireMouseEvent = dll.NewProc("mbFireMouseEvent")
	facade.mbFireMouseWheelEvent = dll.NewProc("mbFireMouseWheelEvent")
	facade.mbFireKeyUpEvent = dll.NewProc("mbFireKeyUpEvent")
	facade.mbFireKeyDownEvent = dll.NewProc("mbFireKeyDownEvent")
	facade.mbFireKeyPressEvent = dll.NewProc("mbFireKeyPressEvent")
	facade.mbFireWindowsMessage = dll.NewProc("mbFireWindowsMessage")

	facade.mbSetCookie = dll.NewProc("mbSetCookie")
	facade.mbGetCookieW = dll.NewProc("mbGetCookie")
	facade.mbVisitAllCookie = dll.NewProc("mbVisitAllCookie")
	facade.mbPerformCookieCommand = dll.NewProc("mbPerformCookieCommand")
	facade.mbClearCookie = dll.NewProc("mbClearCookie")
	facade.mbSetCookieEnabled = dll.NewProc("mbSetCookieEnabled")
	facade.mbSetCookieJarPath = dll.NewProc("mbSetCookieJarPath")
	facade.mbSetCookieJarFullPath = dll.NewProc("mbSetCookieJarFullPath")
	facade.mbSetLocalStorageFullPath = dll.NewProc("mbSetLocalStorageFullPath")
	facade.mbSetDiskCacheEnabled = dll.NewProc("mbSetDiskCacheEnabled")

	facade.mbEditorSelectAll = dll.NewProc("mbEditorSelectAll")
	facade.mbEditorUnSelect = dll.NewProc("mbEditorUnSelect")
	facade.mbEditorCopy = dll.NewProc("mbEditorCopy")
	facade.mbEditorCut = dll.NewProc("mbEditorCut")
	facade.mbEditorDelete = dll.NewProc("mbEditorDelete")
	facade.mbEditorUndo = dll.NewProc("mbEditorUndo")
	facade.mbEditorRedo = dll.NewProc("mbEditorRedo")

	facade.mbRunJS = dll.NewProc("mbRunJs")
	facade.mbToDouble = dll.NewProc("mbJsToDouble")
	facade.mbToString = dll.NewProc("mbJsToString")
	facade.mbGetString = dll.NewProc("mbGetString")
	facade.mbGetStringW = dll.NewProc("mbGetStringW")
	facade.mbSetString = dll.NewProc("mbSetString")
	facade.mbDeleteString = dll.NewProc("mbDeleteString")

	facade.mbPostToUiThread = dll.NewProc("mbPostToUiThread")

	facade.mbSetDiskCachePath = dll.NewProc("mbSetDiskCachePath")
	facade.mbSetDiskCacheLimit = dll.NewProc("mbSetDiskCacheLimit")
	facade.mbSetDiskCacheLimitDisk = dll.NewProc("mbSetDiskCacheLimitDisk")
	facade.mbSetDiskCacheLevel = dll.NewProc("mbSetDiskCacheLevel")

	facade.mbOnTitleChanged = dll.NewProc("mbOnTitleChanged")
	facade.mbOnURLChanged = dll.NewProc("mbOnURLChanged")
	facade.mbOnNavigation = dll.NewProc("mbOnNavigation")
	facade.mbOnDocumentReady = dll.NewProc("mbOnDocumentReady")
	facade.mbOnLoadUrlBegin = dll.NewProc("mbOnLoadUrlBegin")
	facade.mbOnLoadUrlEnd = dll.NewProc("mbOnLoadUrlEnd")
	facade.mbOnJsQuery = dll.NewProc("mbOnJsQuery")
	facade.mbResponseQuery = dll.NewProc("mbResponseQuery")
	facade.mbOnCreateView = dll.NewProc("mbOnCreateView")
	facade.mbOnClose = dll.NewProc("mbOnClose")
	facade.mbOnDestroy = dll.NewProc("mbOnDestroy")
	facade.mbOnAlertBox = dll.NewProc("mbOnAlertBox")
	facade.mbOnConfirmBox = dll.NewProc("mbOnConfirmBox")
	facade.mbOnPromptBox = dll.NewProc("mbOnPromptBox")

	facade.mbNetSetHTTPHeaderField = dll.NewProc("mbNetSetHTTPHeaderField")
	facade.mbNetGetRawHttpHead = dll.NewProc("mbNetGetRawHttpHeadInBlinkThread")
	facade.mbNetSetMIMEType = dll.NewProc("mbNetSetMIMEType")
	facade.mbNetGetMIMEType = dll.NewProc("mbNetGetMIMEType")
	facade.mbNetSetData = dll.NewProc("mbNetSetData")
	facade.mbNetHookRequest = dll.NewProc("mbNetHookRequest")
	facade.mbNetHoldJobToAsynCommit = dll.NewProc("mbNetHoldJobToAsynCommit")
	facade.mbNetContinueJob = dll.NewProc("mbNetContinueJob")
	facade.mbNetGetRequestMethod = dll.NewProc("mbNetGetRequestMethod")
	facade.mbNetGetPostBody = dll.NewProc("mbNetGetPostBody")

	required := map[string]*syscall.LazyProc{
		"mbInit":            facade.mbInit,
		"mbCreateWebWindow": facade.mbCreateWebWindow,
		"mbMoveToCenter":    facade.mbMoveToCenter,
		"mbShowWindow":      facade.mbShowWindow,
		"mbLoadURL":         facade.mbLoadURL,
	}

	for name, proc := range required {
		if err := proc.Find(); err != nil {
			return nil, ProcNotFoundError{Name: name}
		}
	}

	_ = facade.mbRunMessageLoop.Find()
	_ = facade.mbSetDebugConfig.Find()
	_ = facade.wkeSetDebugConfig.Find()
	_ = facade.mbCreateWebView.Find()
	_ = facade.mbDestroyWebView.Find()
	_ = facade.mbDestroyWebWindow.Find()
	_ = facade.mbResize.Find()
	_ = facade.mbGetHostHWND.Find()
	_ = facade.mbSetWindowTitle.Find()
	_ = facade.mbLoadHtmlWithBaseUrl.Find()
	_ = facade.mbGetURL.Find()
	_ = facade.mbStopLoading.Find()
	_ = facade.mbReload.Find()
	_ = facade.mbGoBack.Find()
	_ = facade.mbGoForward.Find()
	_ = facade.mbSetHandle.Find()
	_ = facade.mbSetHandleOffset.Find()
	_ = facade.mbWebFrameGetMainFrame.Find()
	_ = facade.mbIsMainFrame.Find()
	_ = facade.mbSetCspCheckEnable.Find()
	_ = facade.mbSetHeadlessEnabled.Find()
	_ = facade.mbSetNpapiPluginsEnabled.Find()
	_ = facade.mbSetMemoryCacheEnable.Find()
	_ = facade.mbSetProxy.Find()
	_ = facade.mbSetViewProxy.Find()
	_ = facade.mbSetNavigationToNewWindowEnable.Find()
	_ = facade.mbSetUserAgent.Find()
	_ = facade.mbSetZoomFactor.Find()
	_ = facade.mbGetZoomFactor.Find()
	_ = facade.mbSetFocus.Find()
	_ = facade.mbKillFocus.Find()
	_ = facade.mbFireMouseEvent.Find()
	_ = facade.mbFireMouseWheelEvent.Find()
	_ = facade.mbFireKeyUpEvent.Find()
	_ = facade.mbFireKeyDownEvent.Find()
	_ = facade.mbFireKeyPressEvent.Find()
	_ = facade.mbFireWindowsMessage.Find()
	_ = facade.mbSetCookie.Find()
	_ = facade.mbGetCookieW.Find()
	_ = facade.mbVisitAllCookie.Find()
	_ = facade.mbPerformCookieCommand.Find()
	_ = facade.mbClearCookie.Find()
	_ = facade.mbSetCookieEnabled.Find()
	_ = facade.mbSetCookieJarPath.Find()
	_ = facade.mbSetCookieJarFullPath.Find()
	_ = facade.mbSetLocalStorageFullPath.Find()
	_ = facade.mbSetDiskCacheEnabled.Find()
	_ = facade.mbEditorSelectAll.Find()
	_ = facade.mbEditorUnSelect.Find()
	_ = facade.mbEditorCopy.Find()
	_ = facade.mbEditorCut.Find()
	_ = facade.mbEditorDelete.Find()
	_ = facade.mbEditorUndo.Find()
	_ = facade.mbEditorRedo.Find()
	_ = facade.mbRunJS.Find()
	_ = facade.mbToDouble.Find()
	_ = facade.mbToString.Find()
	_ = facade.mbGetString.Find()
	_ = facade.mbGetStringW.Find()
	_ = facade.mbSetString.Find()
	_ = facade.mbDeleteString.Find()
	_ = facade.mbPostToUiThread.Find()
	_ = facade.mbSetDiskCachePath.Find()
	_ = facade.mbSetDiskCacheLimit.Find()
	_ = facade.mbSetDiskCacheLimitDisk.Find()
	_ = facade.mbSetDiskCacheLevel.Find()
	_ = facade.mbOnTitleChanged.Find()
	_ = facade.mbOnURLChanged.Find()
	_ = facade.mbOnNavigation.Find()
	_ = facade.mbOnDocumentReady.Find()
	_ = facade.mbOnLoadUrlBegin.Find()
	_ = facade.mbOnLoadUrlEnd.Find()
	_ = facade.mbOnJsQuery.Find()
	_ = facade.mbResponseQuery.Find()
	_ = facade.mbOnCreateView.Find()
	_ = facade.mbOnClose.Find()
	_ = facade.mbOnDestroy.Find()
	_ = facade.mbOnAlertBox.Find()
	_ = facade.mbOnConfirmBox.Find()
	_ = facade.mbOnPromptBox.Find()
	_ = facade.mbNetSetHTTPHeaderField.Find()
	_ = facade.mbNetGetRawHttpHead.Find()
	_ = facade.mbNetSetMIMEType.Find()
	_ = facade.mbNetGetMIMEType.Find()
	_ = facade.mbNetSetData.Find()
	_ = facade.mbNetHookRequest.Find()
	_ = facade.mbNetHoldJobToAsynCommit.Find()
	_ = facade.mbNetContinueJob.Find()
	_ = facade.mbNetGetRequestMethod.Find()
	_ = facade.mbNetGetPostBody.Find()

	return facade, nil
}
