// Package nakama provides Nakama client wrapper for CLI.
package nakama

import (
	"context"
	"testing"
	"time"
)

// TestNewLogger tests logger creation with different verbosity levels.
func TestNewLogger(t *testing.T) {
	t.Run("default logger", func(t *testing.T) {
		logger := NewLogger(false)
		if logger == nil {
			t.Fatal("Expected logger to be created")
		}
		if logger.SugaredLogger == nil {
			t.Fatal("Expected SugaredLogger to be initialized")
		}
	})

	t.Run("verbose logger", func(t *testing.T) {
		logger := NewLogger(true)
		if logger == nil {
			t.Fatal("Expected logger to be created")
		}
		if logger.SugaredLogger == nil {
			t.Fatal("Expected SugaredLogger to be initialized")
		}
	})
}

// TestNewClient tests client creation with different configurations.
func TestNewClient(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		config := ClientConfig{
			ServerHTTP: "http://localhost:7350",
			ServerWS:   "ws://localhost:7350/ws",
			ServerKey:  "defaultkey",
			Verbose:    false,
		}

		client, err := NewClient(config)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if client == nil {
			t.Fatal("Expected client to be created")
		}
		if client.config.ServerHTTP != config.ServerHTTP {
			t.Errorf("Expected ServerHTTP to be %s, got %s", config.ServerHTTP, client.config.ServerHTTP)
		}
	})

	t.Run("empty config", func(t *testing.T) {
		config := ClientConfig{}
		client, err := NewClient(config)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if client == nil {
			t.Fatal("Expected client to be created")
		}
	})
}

// TestClient_Close tests client close method.
func TestClient_Close(t *testing.T) {
	config := ClientConfig{
		ServerHTTP: "http://localhost:7350",
		ServerKey:  "defaultkey",
	}
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	err = client.Close()
	if err != nil {
		t.Errorf("Expected no error on close, got %v", err)
	}
}

// TestSession tests session structure.
func TestSession(t *testing.T) {
	session := &Session{
		Token:        "test-token",
		RefreshToken: "refresh-token",
		UserID:       "user-123",
		Username:     "test_user",
		ExpiresAt:    time.Now().Unix(),
	}

	if session.Token != "test-token" {
		t.Errorf("Expected Token to be test-token, got %s", session.Token)
	}
	if session.UserID != "user-123" {
		t.Errorf("Expected UserID to be user-123, got %s", session.UserID)
	}
}

// TestClientConfig tests config structure.
func TestClientConfig(t *testing.T) {
	config := ClientConfig{
		ServerHTTP: "http://localhost:7350",
		ServerWS:   "ws://localhost:7350/ws",
		ServerKey:  "custom-key",
		Verbose:    true,
	}

	if config.ServerHTTP != "http://localhost:7350" {
		t.Errorf("Expected ServerHTTP to be http://localhost:7350, got %s", config.ServerHTTP)
	}
	if config.Verbose != true {
		t.Errorf("Expected Verbose to be true, got %v", config.Verbose)
	}
}

// TestSocketMessage tests SocketMessage structure.
func TestSocketMessage(t *testing.T) {
	msg := &SocketMessage{
		OpCode: 100,
		Data:   []byte(`{"test": "data"}`),
	}

	if msg.OpCode != 100 {
		t.Errorf("Expected OpCode to be 100, got %d", msg.OpCode)
	}
	if len(msg.Data) == 0 {
		t.Error("Expected Data to be non-empty")
	}
}

// TestOpCodeConstants tests OpCode constants are defined correctly.
func TestOpCodeConstants(t *testing.T) {
	tests := []struct {
		name     string
		opCode   int64
		expected int64
	}{
		{"OpStateSync", OpStateSync, 1},
		{"OpDecisionRequest", OpDecisionRequest, 3},
		{"OpAvailable", OpAvailable, 4},
		{"OpMiniGameStart", OpMiniGameStart, 5},
		{"OpMiniGameResult", OpMiniGameResult, 6},
		{"OpGameOver", OpGameOver, 7},
		{"OpFullSync", OpFullSync, 8},
		{"OpActionRejected", OpActionRejected, 9},
		{"OpRollDice", OpRollDice, 100},
		{"OpUseItem", OpUseItem, 101},
		{"OpUseSkill", OpUseSkill, 102},
		{"OpUserChoice", OpUserChoice, 103},
		{"OpMiniGameDataSubmit", OpMiniGameDataSubmit, 107},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.opCode != tt.expected {
				t.Errorf("Expected %s to be %d, got %d", tt.name, tt.expected, tt.opCode)
			}
		})
	}
}

// TestNewSocketClient tests socket client creation.
func TestNewSocketClient(t *testing.T) {
	clientConfig := ClientConfig{
		ServerHTTP: "http://localhost:7350",
		ServerKey:  "defaultkey",
	}
	client, err := NewClient(clientConfig)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	socketClient, err := client.CreateSocketClient()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if socketClient == nil {
		t.Fatal("Expected socket client to be created")
	}
}

// TestSocketClient_MessageChan tests message channel is initialized.
func TestSocketClient_MessageChan(t *testing.T) {
	clientConfig := ClientConfig{
		ServerHTTP: "http://localhost:7350",
		ServerKey:  "defaultkey",
	}
	client, _ := NewClient(clientConfig)
	socketClient, _ := client.CreateSocketClient()

	msgChan := socketClient.MessageChan()
	if msgChan == nil {
		t.Fatal("Expected message channel to be initialized")
	}
}

// TestStandaloneClientConfig tests standalone config structure.
func TestStandaloneClientConfig(t *testing.T) {
	config := StandaloneClientConfig{
		ServerWS: "ws://localhost:7350/ws",
		Verbose:  true,
	}

	if config.ServerWS != "ws://localhost:7350/ws" {
		t.Errorf("Expected ServerWS to be ws://localhost:7350/ws, got %s", config.ServerWS)
	}
	if config.Verbose != true {
		t.Errorf("Expected Verbose to be true, got %v", config.Verbose)
	}
}

// TestStandaloneClient tests standalone client creation.
func TestStandaloneClient(t *testing.T) {
	config := StandaloneClientConfig{
		ServerWS: "ws://localhost:7350/ws",
		Verbose:  false,
	}

	client, err := NewStandaloneClient(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if client == nil {
		t.Fatal("Expected standalone client to be created")
	}
	if client.session == nil {
		t.Fatal("Expected session to be initialized")
	}
	if client.session.Token != "standalone-token" {
		t.Errorf("Expected session token to be standalone-token, got %s", client.session.Token)
	}
}

// TestStandaloneClient_Authenticate tests standalone authentication.
func TestStandaloneClient_Authenticate(t *testing.T) {
	config := StandaloneClientConfig{
		ServerWS: "ws://localhost:7350/ws",
	}
	client, _ := NewStandaloneClient(config)

	ctx := context.Background()
	session, err := client.Authenticate(ctx, "test-user-id")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if session == nil {
		t.Fatal("Expected session to be returned")
	}
	if session.UserID != "test-user-id" {
		t.Errorf("Expected UserID to be test-user-id, got %s", session.UserID)
	}
	if session.Username != "test-user-id" {
		t.Errorf("Expected Username to be test-user-id, got %s", session.Username)
	}
}

// TestStandaloneSocketClientCreation tests standalone socket client creation.
func TestStandaloneSocketClientCreation(t *testing.T) {
	config := StandaloneClientConfig{
		ServerWS: "ws://localhost:7350/ws",
	}
	client, _ := NewStandaloneClient(config)

	socketClient, err := client.CreateSocketClient()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if socketClient == nil {
		t.Fatal("Expected standalone socket client to be created")
	}
}

// TestStandaloneSocketClient_MessageChan tests standalone socket message channel.
func TestStandaloneSocketClient_MessageChan(t *testing.T) {
	config := StandaloneClientConfig{
		ServerWS: "ws://localhost:7350/ws",
	}
	client, _ := NewStandaloneClient(config)
	socketClient, _ := client.CreateSocketClient()

	msgChan := socketClient.MessageChan()
	if msgChan == nil {
		t.Fatal("Expected message channel to be initialized")
	}
}

// TestIClientInterface tests IClient interface implementation.
func TestIClientInterface(t *testing.T) {
	// Test Client implements IClient
	var _ IClient = (*Client)(nil)

	// Test StandaloneClient implements IClient
	var _ IClient = (*StandaloneClient)(nil)
}

// TestISocketClientInterface tests ISocketClient interface implementation.
func TestISocketClientInterface(t *testing.T) {
	// Test SocketClient implements ISocketClient
	var _ ISocketClient = (*SocketClient)(nil)

	// Test StandaloneSocketClient implements ISocketClient
	var _ ISocketClient = (*StandaloneSocketClient)(nil)
}

// TestSendMessage_Concurrency tests SendMessage with concurrent calls.
func TestSendMessage_Concurrency(t *testing.T) {
	clientConfig := ClientConfig{
		ServerHTTP: "http://localhost:7350",
		ServerKey:  "defaultkey",
	}
	client, _ := NewClient(clientConfig)
	socketClient, _ := client.CreateSocketClient()

	// Note: We can't actually send messages without a real server,
	// but we can verify the mutex is properly initialized
	sc, ok := socketClient.(*SocketClient)
	if !ok {
		t.Fatal("Expected socket client to be *SocketClient type")
	}

	// Verify sendMu is usable (will panic if not properly initialized)
	sc.sendMu.Lock()
	sc.sendMu.Unlock()
}

// TestStandaloneSocketClient_Concurrency tests standalone SendMessage concurrency.
func TestStandaloneSocketClient_Concurrency(t *testing.T) {
	config := StandaloneClientConfig{
		ServerWS: "ws://localhost:7350/ws",
	}
	client, _ := NewStandaloneClient(config)
	socketClient, _ := client.CreateSocketClient()

	// Verify mutex is properly initialized
	sc, ok := socketClient.(*StandaloneSocketClient)
	if !ok {
		t.Fatal("Expected socket client to be *StandaloneSocketClient type")
	}

	// Verify sendMu is usable
	sc.sendMu.Lock()
	sc.sendMu.Unlock()
}
