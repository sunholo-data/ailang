package apiserver

import (
	"net"

	"github.com/sunholo-data/ailang/internal/config"
)

// bindHost is the host serve-api listens on: Config.Bind when given,
// otherwise config.DefaultBindHost() (127.0.0.1, or 0.0.0.0 when PORT is set).
func (s *Server) bindHost() string {
	if s.bind != "" {
		return s.bind
	}
	return config.DefaultBindHost()
}

// listen binds the HTTP listener eagerly, so a taken port fails Start before
// the startup banner prints (M-SERVEAPI-BIND-HOST-CORS M1). JoinHostPort
// brackets IPv6 hosts such as ::1.
func (s *Server) listen() (net.Listener, error) {
	return net.Listen("tcp", net.JoinHostPort(s.bindHost(), s.port))
}
