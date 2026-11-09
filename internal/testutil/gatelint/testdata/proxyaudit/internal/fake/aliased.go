package fake

import nethttp "net/http"

func aliased() nethttp.RoundTripper {
	return &nethttp.Transport{ // finding: aliased import, multi-line, no Proxy
		MaxIdleConns: 1,
	}
}
