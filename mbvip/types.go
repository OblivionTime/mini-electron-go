package mbvip

import "runtime"

type WebView uintptr
type WebFrameHandle uintptr
type MbString uintptr

type WindowType int

const (
	WindowTypePopup WindowType = iota
	WindowTypeTransparent
	WindowTypeControl
)

type WindowOptions struct {
	Type   WindowType
	Parent uintptr
	X      int
	Y      int
	Width  int
	Height int
}

type Config struct {
	DLLPath string
}

type ProxyType int32

const (
	ProxyNone           ProxyType = 0
	ProxyHTTP           ProxyType = 1
	ProxySOCKS4         ProxyType = 2
	ProxySOCKS4A        ProxyType = 3
	ProxySOCKS5         ProxyType = 4
	ProxySOCKS5HostName ProxyType = 5
)

type Proxy struct {
	Type     ProxyType
	Hostname string
	Port     uint16
	Username string
	Password string
}

type NavigationType int32

const (
	NavigationTypeLinkClick   NavigationType = 0
	NavigationTypeFormSubmit  NavigationType = 1
	NavigationTypeBackForward NavigationType = 2
	NavigationTypeReload      NavigationType = 3
)

type RequestType int32

const (
	RequestTypeGet  RequestType = 1
	RequestTypePost RequestType = 2
)

type CookieCommand int32

const (
	CookieCommandClearAllCookies       CookieCommand = 0
	CookieCommandClearSessionCookies   CookieCommand = 1
	CookieCommandFlushCookiesToFile    CookieCommand = 2
	CookieCommandReloadCookiesFromFile CookieCommand = 3
)

type mbProxy struct {
	Type     int32
	Hostname [100]byte
	Port     uint16
	Username [50]byte
	Password [50]byte
}

type Client struct {
	dll    *dllFacade
	id     uintptr
	events *eventHub
}

func NewClient(cfg Config) (*Client, error) {
	if runtime.GOOS != "windows" {
		return nil, ErrUnsupportedPlatform
	}

	dllPath := cfg.DLLPath
	if dllPath == "" {
		dllPath = "mb132_x64.dll"
	}

	dll, err := loadDLL(dllPath)
	if err != nil {
		return nil, err
	}

	client := &Client{
		dll:    dll,
		id:     nextClientID(),
		events: newEventHub(),
	}
	registerClient(client)
	return client, nil
}
