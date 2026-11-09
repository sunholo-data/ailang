// Package fake is the ProxyAudit known-positive fixture: three nil-Proxy
// literals the audit must report and two clean ones it must not.
package fake

import (
	"net/http"
	"time"
)

var bare = &http.Transport{} // finding: no Proxy field

func explicitNil() *http.Transport {
	return &http.Transport{ // finding: Proxy: nil
		Proxy:           nil,
		IdleConnTimeout: 90 * time.Second,
	}
}

func clean() *http.Client {
	return &http.Client{Transport: &http.Transport{Proxy: http.ProxyFromEnvironment}}
}

func pinnedProxy() *http.Transport {
	return &http.Transport{Proxy: http.ProxyURL(nil)}
}
