//go:build !windows

package update

import (
	"net/http"
	"net/url"
)

// proxyFor is the environment's proxy off Windows (see proxy_windows.go).
func proxyFor(req *http.Request) (*url.URL, error) { return http.ProxyFromEnvironment(req) }

func proxyScriptOnly() bool { return false }
