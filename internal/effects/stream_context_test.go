package effects

import (
	"testing"
)

func TestNewStreamContext_Defaults(t *testing.T) {
	sc := NewStreamContext()

	if sc.MaxConnections != 4 {
		t.Errorf("MaxConnections = %d, want 4", sc.MaxConnections)
	}
	if sc.MaxMessageSize != 1*1024*1024 {
		t.Errorf("MaxMessageSize = %d, want 1MB", sc.MaxMessageSize)
	}
	if sc.MaxFrameSize != 64*1024 {
		t.Errorf("MaxFrameSize = %d, want 64KB", sc.MaxFrameSize)
	}
	if sc.AllowHTTP {
		t.Error("AllowHTTP should be false by default")
	}
	if sc.AllowLocalhost {
		t.Error("AllowLocalhost should be false by default")
	}
	if !sc.BlockPrivateIPs {
		t.Error("BlockPrivateIPs should be true by default")
	}
	if sc.EventBufferSize != 1000 {
		t.Errorf("EventBufferSize = %d, want 1000", sc.EventBufferSize)
	}
}

func TestStreamContext_ValidateURL_ProtocolCheck(t *testing.T) {
	sc := NewStreamContext()

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"wss allowed", "wss://example.com/ws", false},
		{"https allowed", "https://example.com/ws", false},
		{"ws blocked by default", "ws://example.com/ws", true},
		{"http blocked by default", "http://example.com/ws", true},
		{"ftp blocked", "ftp://example.com/ws", true},
		{"empty scheme blocked", "://example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sc.ValidateURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestStreamContext_ValidateURL_AllowHTTP(t *testing.T) {
	sc := NewStreamContext()
	sc.AllowHTTP = true

	if err := sc.ValidateURL("ws://example.com/ws"); err != nil {
		t.Errorf("ws:// should be allowed with AllowHTTP=true: %v", err)
	}
	if err := sc.ValidateURL("http://example.com/ws"); err != nil {
		t.Errorf("http:// should be allowed with AllowHTTP=true: %v", err)
	}
}

func TestStreamContext_ValidateURL_DomainAllowlist(t *testing.T) {
	sc := NewStreamContext()
	sc.AllowedDomains = []string{"example.com", "*.test.io"}

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"exact match allowed", "wss://example.com/ws", false},
		{"wildcard match allowed", "wss://sub.test.io/ws", false},
		{"not in allowlist", "wss://evil.com/ws", true},
		{"partial match blocked", "wss://notexample.com/ws", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sc.ValidateURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestStreamContext_ValidateURL_LocalhostBlocking(t *testing.T) {
	sc := NewStreamContext()

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"localhost blocked", "wss://localhost/ws", true},
		{"127.0.0.1 blocked", "wss://127.0.0.1/ws", true},
		{"127.0.0.2 blocked", "wss://127.0.0.2/ws", true},
		{"::1 blocked", "wss://[::1]/ws", true},
		{"external allowed", "wss://example.com/ws", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sc.ValidateURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestStreamContext_ValidateURL_AllowLocalhost(t *testing.T) {
	sc := NewStreamContext()
	sc.AllowLocalhost = true

	if err := sc.ValidateURL("wss://localhost/ws"); err != nil {
		t.Errorf("localhost should be allowed with AllowLocalhost=true: %v", err)
	}
}

func TestStreamContext_ValidateURL_PrivateIPBlocking(t *testing.T) {
	sc := NewStreamContext()

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"10.x blocked", "wss://10.0.0.1/ws", true},
		{"172.16.x blocked", "wss://172.16.0.1/ws", true},
		{"192.168.x blocked", "wss://192.168.1.1/ws", true},
		{"169.254.x blocked", "wss://169.254.0.1/ws", true},
		{"public IP allowed", "wss://8.8.8.8/ws", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sc.ValidateURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestStreamContext_ValidateURL_MissingHostname(t *testing.T) {
	sc := NewStreamContext()
	err := sc.ValidateURL("wss:///path")
	if err == nil {
		t.Error("expected error for URL with missing hostname")
	}
}

func TestStreamContext_ConnectionLimit(t *testing.T) {
	sc := NewStreamContext()
	sc.MaxConnections = 2

	conn1 := &StreamConnection{status: StreamStatusOpen}
	conn2 := &StreamConnection{status: StreamStatusOpen}
	conn3 := &StreamConnection{status: StreamStatusOpen}

	id1, err := sc.AcquireConnection(conn1)
	if err != nil {
		t.Fatalf("AcquireConnection #1 failed: %v", err)
	}
	if id1 != 1 {
		t.Errorf("first ID = %d, want 1", id1)
	}

	id2, err := sc.AcquireConnection(conn2)
	if err != nil {
		t.Fatalf("AcquireConnection #2 failed: %v", err)
	}
	if id2 != 2 {
		t.Errorf("second ID = %d, want 2", id2)
	}

	// Third should fail
	_, err = sc.AcquireConnection(conn3)
	if err == nil {
		t.Error("expected error when exceeding MaxConnections")
	}

	if sc.ConnectionCount() != 2 {
		t.Errorf("ConnectionCount = %d, want 2", sc.ConnectionCount())
	}

	// Release one and try again
	sc.ReleaseConnection(id1)
	if sc.ConnectionCount() != 1 {
		t.Errorf("ConnectionCount after release = %d, want 1", sc.ConnectionCount())
	}

	id3, err := sc.AcquireConnection(conn3)
	if err != nil {
		t.Fatalf("AcquireConnection #3 after release failed: %v", err)
	}
	if id3 != 3 {
		t.Errorf("third ID = %d, want 3", id3)
	}
}

func TestStreamContext_GetConnection(t *testing.T) {
	sc := NewStreamContext()
	conn := &StreamConnection{status: StreamStatusOpen}

	id, _ := sc.AcquireConnection(conn)

	got, ok := sc.GetConnection(id)
	if !ok {
		t.Fatal("GetConnection returned false for existing connection")
	}
	if got != conn {
		t.Error("GetConnection returned different connection")
	}

	_, ok = sc.GetConnection(999)
	if ok {
		t.Error("GetConnection returned true for non-existent connection")
	}
}

func TestStreamContext_CloseAll(t *testing.T) {
	sc := NewStreamContext()

	// Create connections with done channels (no actual websocket)
	conn1 := &StreamConnection{
		status: StreamStatusOpen,
		done:   make(chan struct{}),
	}
	conn2 := &StreamConnection{
		status: StreamStatusOpen,
		done:   make(chan struct{}),
	}

	sc.AcquireConnection(conn1)
	sc.AcquireConnection(conn2)

	sc.CloseAll()

	if conn1.Status() != StreamStatusClosed {
		t.Errorf("conn1 status = %v, want Closed", conn1.Status())
	}
	if conn2.Status() != StreamStatusClosed {
		t.Errorf("conn2 status = %v, want Closed", conn2.Status())
	}
}

func TestIsLocalhost(t *testing.T) {
	tests := []struct {
		host string
		want bool
	}{
		{"localhost", true},
		{"LOCALHOST", true},
		{"127.0.0.1", true},
		{"127.0.0.2", true},
		{"::1", true},
		{"example.com", false},
		{"10.0.0.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			if got := isLocalhost(tt.host); got != tt.want {
				t.Errorf("isLocalhost(%q) = %v, want %v", tt.host, got, tt.want)
			}
		})
	}
}

func TestIsStreamAllowedDomain(t *testing.T) {
	allowed := []string{"example.com", "*.test.io"}

	tests := []struct {
		host string
		want bool
	}{
		{"example.com", true},
		{"Example.Com", true},
		{"sub.test.io", true},
		{"deep.sub.test.io", true},
		{"test.io", false},
		{"evil.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			if got := isStreamAllowedDomain(tt.host, allowed); got != tt.want {
				t.Errorf("isStreamAllowedDomain(%q) = %v, want %v", tt.host, got, tt.want)
			}
		})
	}
}

func TestStreamContext_RegistryEntry(t *testing.T) {
	// Verify Stream is in the Registry
	ops, ok := Registry["Stream"]
	if !ok {
		t.Fatal("Stream not found in Registry")
	}

	expectedOps := []string{"connect", "send", "onEvent", "runEventLoop", "close", "status"}
	for _, name := range expectedOps {
		if _, exists := ops[name]; !exists {
			t.Errorf("Stream operation %q not registered", name)
		}
	}
}
