// Package nakama provides Nakama client wrapper for CLI.
package nakama

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ascii8/nakama-go"
	pkgnet "github.com/b1tAction/paradiced/pkg/net"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"nhooyr.io/websocket"
)

// Logger wraps zap logger for CLI usage.
type Logger struct {
	*zap.SugaredLogger
}

// NewLogger creates a new logger with the given verbosity.
func NewLogger(verbose bool) *Logger {
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	if verbose {
		config.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	logger, err := config.Build()
	if err != nil {
		panic(err)
	}

	return &Logger{logger.Sugar()}
}

// ClientConfig holds configuration for Nakama client.
type ClientConfig struct {
	ServerHTTP string
	ServerWS   string
	ServerKey  string
	Verbose    bool
}

// Client wraps Nakama client for CLI usage.
type Client struct {
	*nakama.Client
	config     ClientConfig
	logger     *Logger
	httpClient *http.Client
}

// Ensure Client implements IClient
var _ IClient = (*Client)(nil)

// CreateSocketClient creates a new socket client.
func (c *Client) CreateSocketClient() (ISocketClient, error) {
	return NewSocketClient(c)
}

// NewClient creates a new Nakama client.
func NewClient(config ClientConfig) (*Client, error) {
	logger := NewLogger(config.Verbose)

	// Create HTTP client
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create nakama-go client
	client := nakama.New(
		nakama.WithURL(config.ServerHTTP),
		nakama.WithServerKey(config.ServerKey),
		nakama.WithLogger(func(format string, v ...interface{}) {
			logger.Debugf(format, v...)
		}),
		nakama.WithHttpClient(httpClient),
	)

	return &Client{
		Client:     client,
		config:     config,
		logger:     logger,
		httpClient: httpClient,
	}, nil
}

// Close closes the client.
func (c *Client) Close() error {
	return nil
}

// Authenticate authenticates a user and returns the session.
func (c *Client) Authenticate(ctx context.Context, userID string) (*Session, error) {
	// Use device authentication
	err := c.Client.AuthenticateDevice(ctx, userID, true, userID)
	if err != nil {
		return nil, err
	}

	// Get account info
	account, err := c.Client.Account(ctx)
	if err != nil {
		return nil, err
	}

	c.logger.Debug("Authentication successful", "user_id", account.User.Id)
	return &Session{
		Token:        c.Client.SessionToken(),
		RefreshToken: c.Client.SessionRefreshToken(),
		UserID:       account.User.Id,
		Username:     account.User.Username,
		ExpiresAt:    c.Client.SessionExpiry().Unix(),
	}, nil
}

// SocketClient wraps Nakama socket connection.
type SocketClient struct {
	conn      *nakama.Conn
	session   *Session
	matchID   string
	logger    *Logger
	msgChan   chan *SocketMessage
	closeChan chan struct{}
	config    ClientConfig
	sendMu    sync.Mutex // Protects concurrent MatchDataSend calls
}

// Ensure SocketClient implements ISocketClient
var _ ISocketClient = (*SocketClient)(nil)

// SocketMessage represents a received match message.
type SocketMessage struct {
	OpCode int64
	Data   []byte
}

// NewSocketClient creates a new socket client.
func NewSocketClient(client *Client) (*SocketClient, error) {
	return &SocketClient{
		logger:    client.logger,
		msgChan:   make(chan *SocketMessage, 100),
		closeChan: make(chan struct{}),
		config:    client.config,
	}, nil
}

// Connect establishes the WebSocket connection.
func (sc *SocketClient) Connect(ctx context.Context, session *Session) error {
	sc.session = session

	// Build WS URL
	wsURL := sc.config.ServerWS
	if wsURL == "" {
		// Default to replacing http:// with ws://
		wsURL = sc.config.ServerHTTP
	}

	// Replace http:// with ws://
	if len(wsURL) >= 7 && wsURL[:7] == "http://" {
		wsURL = "ws://" + wsURL[7:] + "/ws"
	} else if len(wsURL) >= 8 && wsURL[:8] == "https://" {
		wsURL = "wss://" + wsURL[8:] + "/ws"
	} else if !strings.HasSuffix(wsURL, "/ws") {
		wsURL = wsURL + "/ws"
	}

	sc.logger.Debug("Connecting to WebSocket", "url", wsURL)

	// Create connection using nakama-go with client handler
	conn, err := nakama.NewConn(ctx,
		nakama.WithConnUrl(wsURL),
		nakama.WithConnToken(session.Token),
		nakama.WithConnClientHandler(&connClientHandler{
			logger:     sc.logger,
			httpClient: &http.Client{Timeout: 30 * time.Second},
			socketURL:  wsURL,
			token:      session.Token,
			sessionEnd: func() { sc.logger.Info("Session ended") },
		}),
	)
	if err != nil {
		return err
	}

	// Set up handlers before opening
	conn.MatchDataHandler = func(ctx context.Context, msg *nakama.MatchDataMsg) {
		if msg == nil {
			return
		}

		// Skip own echoed messages when sender presence exists.
		// Check msg.Presence first to avoid nil pointer dereference.
		if msg.Presence == nil {
			// System message (e.g., matchmaker matched notification)
			sc.logger.Debug("Received system message (no sender)", "op_code", msg.OpCode)
		} else if sc.session != nil && msg.Presence.UserId == sc.session.UserID {
			return
		}

		sc.logger.Debug("Received match message", "op_code", msg.OpCode)

		select {
		case sc.msgChan <- &SocketMessage{
			OpCode: msg.OpCode,
			Data:   msg.Data,
		}:
		case <-sc.closeChan:
			return
		}
	}

	// Open connection
	if err := conn.Open(ctx); err != nil {
		return err
	}

	sc.conn = conn
	sc.logger.Info("WebSocket connection established")

	return nil
}

// connClientHandler implements nakama.ConnClientHandler.
type connClientHandler struct {
	logger     *Logger
	httpClient *http.Client
	socketURL  string
	token      string
	sessionEnd func()
}

func (h *connClientHandler) HttpClient() *http.Client                  { return h.httpClient }
func (h *connClientHandler) SocketURL() (string, error)                { return h.socketURL, nil }
func (h *connClientHandler) Token(ctx context.Context) (string, error) { return h.token, nil }
func (h *connClientHandler) SessionEnd()                               { h.sessionEnd() }
func (h *connClientHandler) Logf(format string, v ...interface{})      { h.logger.Debugf(format, v...) }
func (h *connClientHandler) Errf(format string, v ...interface{})      { h.logger.Errorf(format, v...) }

// CreateMatch creates a new match by calling MatchCreate with the given name.
// For authoritative matches, this will trigger the match handler's MatchInit.
func (sc *SocketClient) CreateMatch(ctx context.Context, name string) (string, error) {
	match, err := sc.conn.MatchCreate(ctx, name)
	if err != nil {
		return "", err
	}

	sc.matchID = match.MatchId
	sc.logger.Info("Match created successfully", "match_id", sc.matchID)
	return sc.matchID, nil
}

// JoinMatch joins an existing match by ID or token.
func (sc *SocketClient) JoinMatch(ctx context.Context, matchIDOrToken string) error {
	if sc.conn == nil {
		return fmt.Errorf("socket connection not established")
	}

	// Check if it's a token (JWT with dots) or match ID
	// Token format: JWT (e.g., "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...")
	// Match ID format: UUID with dot suffix (e.g., "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx.")
	var match *nakama.MatchMsg
	var err error

	if strings.Count(matchIDOrToken, ".") == 2 {
		// Likely a JWT token (starts with "eyJ")
		sc.logger.Debug("Joining match with token", "token_len", len(matchIDOrToken))
		match, err = sc.conn.MatchJoinToken(ctx, matchIDOrToken, nil)
	} else {
		// Use as match ID
		sc.logger.Debug("Joining match with ID", "match_id", matchIDOrToken)
		match, err = sc.conn.MatchJoin(ctx, matchIDOrToken, nil)
	}

	if err != nil {
		return err
	}

	sc.matchID = match.MatchId
	sc.logger.Info("Joined match successfully", "match_id", sc.matchID)
	return nil
}

// SendMessage sends a message to the match.
func (sc *SocketClient) SendMessage(ctx context.Context, opCode int64, data any) error {
	// Marshal data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	sc.logger.Debug("SendMessage called", "op_code", opCode, "data", string(jsonData), "match_id", sc.matchID)

	// Use mutex to protect concurrent MatchDataSend calls
	sc.sendMu.Lock()
	defer sc.sendMu.Unlock()

	err = sc.conn.MatchDataSend(ctx, sc.matchID, opCode, jsonData, true, nil)
	sc.logger.Debug("MatchDataSend returned", "op_code", opCode, "error", err)
	return err
}

// AddMatchmaker adds self to matchmaker and returns the ticket.
func (sc *SocketClient) AddMatchmaker(ctx context.Context, query string, minPlayers, maxPlayers int, props map[string]string, numericProps map[string]float64) (*nakama.MatchmakerTicketMsg, error) {
	return sc.conn.MatchmakerAdd(ctx, nakama.MatchmakerAdd(query, minPlayers, maxPlayers).
		WithStringProperties(props).
		WithNumericProperties(numericProps))
}

// SetMatchmakerMatchedHandler sets the handler for matchmaker matched events.
func (sc *SocketClient) SetMatchmakerMatchedHandler(handler func(context.Context, *nakama.MatchmakerMatchedMsg)) {
	sc.conn.MatchmakerMatchedHandler = handler
}

// SetMatchID sets the current match ID.
func (sc *SocketClient) SetMatchID(matchID string) {
	sc.matchID = matchID
}

// MatchDataMessage is the type alias for nakama match data message.
type MatchDataMessage = nakama.MatchDataMsg

// MatchmakerTicketMsg is the type alias for nakama matchmaker ticket message.
type MatchmakerTicketMsg = nakama.MatchmakerTicketMsg

// MatchmakerMatchedMsg is the type alias for nakama matchmaker matched message.
type MatchmakerMatchedMsg = nakama.MatchmakerMatchedMsg

// MessageChan returns the message channel for receiving messages.
func (sc *SocketClient) MessageChan() <-chan *SocketMessage {
	return sc.msgChan
}

// Close closes the socket client.
func (sc *SocketClient) Close() error {
	close(sc.closeChan)
	if sc.conn != nil {
		return sc.conn.Close()
	}
	sc.logger.Info("WebSocket connection closed")
	return nil
}

// Session represents a Nakama session.
type Session struct {
	Token        string
	RefreshToken string
	UserID       string
	Username     string
	ExpiresAt    int64
}

// OpCode constants (same as pkg/net/opcode.go)
const (
	// Server -> Client: 1-99
	OpStateSync       int64 = 1
	OpTurnSync        int64 = 2
	OpDecisionRequest int64 = 3
	OpAvailable       int64 = 4
	OpMiniGameStart   int64 = 5
	OpMiniGameResult  int64 = 6
	OpGameOver        int64 = 7
	OpFullSync        int64 = 8
	OpActionRejected  int64 = 9

	// Client -> Server: 100+
	OpRollDice             int64 = 100
	OpUseItem              int64 = 101
	OpUseSkill             int64 = 102
	OpUserChoice           int64 = 103
	OpMiniGameResultSubmit int64 = 104
)

// IClient interface for both Nakama and Standalone clients.
type IClient interface {
	Authenticate(ctx context.Context, userID string) (*Session, error)
	CreateSocketClient() (ISocketClient, error)
	Close() error
}

// StandaloneClientConfig holds configuration for standalone WebSocket client.
type StandaloneClientConfig struct {
	ServerWS string
	Verbose  bool
}

// StandaloneClient implements IClient for standalone WebSocket server.
type StandaloneClient struct {
	config  StandaloneClientConfig
	logger  *Logger
	session *Session
}

// NewStandaloneClient creates a new standalone WebSocket client.
func NewStandaloneClient(config StandaloneClientConfig) (*StandaloneClient, error) {
	logger := NewLogger(config.Verbose)

	// Create a mock session for standalone mode
	session := &Session{
		Token:     "standalone-token",
		UserID:    "", // Will be set per player
		Username:  "",
		ExpiresAt: 0,
	}

	return &StandaloneClient{
		config:  config,
		logger:  logger,
		session: session,
	}, nil
}

// Authenticate creates a mock session for standalone mode.
func (c *StandaloneClient) Authenticate(ctx context.Context, userID string) (*Session, error) {
	c.session.UserID = userID
	c.session.Username = userID
	c.logger.Debug("Authentication successful", "user_id", userID)
	return c.session, nil
}

// CreateSocketClient creates a new standalone socket client.
func (c *StandaloneClient) CreateSocketClient() (ISocketClient, error) {
	return NewStandaloneSocketClient(c.config, c.logger)
}

// Close closes the client.
func (c *StandaloneClient) Close() error {
	return nil
}

// ISocketClient interface for socket clients.
type ISocketClient interface {
	Connect(ctx context.Context, session *Session) error
	CreateMatch(ctx context.Context, name string) (string, error)
	JoinMatch(ctx context.Context, matchIDOrToken string) error
	SendMessage(ctx context.Context, opCode int64, data any) error
	MessageChan() <-chan *SocketMessage
	Close() error
	SetMatchmakerMatchedHandler(handler func(context.Context, *nakama.MatchmakerMatchedMsg))
	SetMatchID(matchID string)
	AddMatchmaker(ctx context.Context, query string, minPlayers, maxPlayers int, props map[string]string, numericProps map[string]float64) (*nakama.MatchmakerTicketMsg, error)
}

// StandaloneSocketClient wraps standalone WebSocket connection.
type StandaloneSocketClient struct {
	conn      *websocket.Conn
	session   *Session
	matchID   string
	logger    *Logger
	msgChan   chan *SocketMessage
	closeChan chan struct{}
	config    StandaloneClientConfig
	sendMu    sync.Mutex // Protects concurrent WebSocket write calls
}

// Ensure StandaloneSocketClient implements ISocketClient
var _ ISocketClient = (*StandaloneSocketClient)(nil)

// NewStandaloneSocketClient creates a new standalone socket client.
func NewStandaloneSocketClient(config StandaloneClientConfig, logger *Logger) (*StandaloneSocketClient, error) {
	return &StandaloneSocketClient{
		logger:    logger,
		msgChan:   make(chan *SocketMessage, 100),
		closeChan: make(chan struct{}),
		config:    config,
	}, nil
}

// Connect establishes the WebSocket connection to standalone server.
func (sc *StandaloneSocketClient) Connect(ctx context.Context, session *Session) error {
	sc.session = session

	// Build WS URL for standalone server
	wsURL := sc.config.ServerWS
	if !strings.HasPrefix(wsURL, "ws://") && !strings.HasPrefix(wsURL, "wss://") {
		wsURL = "ws://" + strings.TrimPrefix(wsURL, "http://")
	}
	if !strings.HasSuffix(wsURL, "/ws") {
		wsURL = wsURL + "/ws"
	}

	// Add user_id parameter
	wsURL = fmt.Sprintf("%s?user_id=%s", wsURL, session.UserID)

	sc.logger.Debug("Connecting to standalone WebSocket", "url", wsURL)

	// Dial WebSocket
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		return err
	}

	sc.conn = conn
	sc.logger.Info("Standalone WebSocket connection established")

	// Start read loop
	go sc.readLoop(ctx)

	return nil
}

// readLoop reads messages from WebSocket.
func (sc *StandaloneSocketClient) readLoop(ctx context.Context) {
	for {
		_, data, err := sc.conn.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
				return
			}
			sc.logger.Error("WebSocket read error", "error", err)
			return
		}

		select {
		case sc.msgChan <- &SocketMessage{
			Data: data,
		}:
		case <-sc.closeChan:
			return
		}
	}
}

// CreateMatch is not used in standalone mode (server handles match creation).
func (sc *StandaloneSocketClient) CreateMatch(ctx context.Context, name string) (string, error) {
	// Standalone mode doesn't use matchmaker
	return "standalone-match", nil
}

// JoinMatch is not used in standalone mode.
func (sc *StandaloneSocketClient) JoinMatch(ctx context.Context, matchIDOrToken string) error {
	sc.matchID = matchIDOrToken
	return nil
}

// SendMessage sends a message to the standalone server.
func (sc *StandaloneSocketClient) SendMessage(ctx context.Context, opCode int64, data any) error {
	// Wrap data in Message format
	msg := pkgnet.Message{
		OpCode:    pkgnet.OpCode(opCode),
		Timestamp: time.Now().UnixMilli(),
	}

	// Marshal data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	msg.Data = jsonData

	// Marshal full message
	messageData, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// Use mutex to protect concurrent WebSocket write calls
	sc.sendMu.Lock()
	defer sc.sendMu.Unlock()

	return sc.conn.Write(ctx, websocket.MessageBinary, messageData)
}

// AddMatchmaker is not used in standalone mode.
func (sc *StandaloneSocketClient) AddMatchmaker(ctx context.Context, query string, minPlayers, maxPlayers int, props map[string]string, numericProps map[string]float64) (*nakama.MatchmakerTicketMsg, error) {
	return nil, fmt.Errorf("matchmaker not available in standalone mode")
}

// SetMatchmakerMatchedHandler is not used in standalone mode.
func (sc *StandaloneSocketClient) SetMatchmakerMatchedHandler(handler func(context.Context, *nakama.MatchmakerMatchedMsg)) {
	// Not used in standalone mode
}

// SetMatchID sets the current match ID.
func (sc *StandaloneSocketClient) SetMatchID(matchID string) {
	sc.matchID = matchID
}

// MessageChan returns the message channel.
func (sc *StandaloneSocketClient) MessageChan() <-chan *SocketMessage {
	return sc.msgChan
}

// Close closes the standalone socket client.
func (sc *StandaloneSocketClient) Close() error {
	close(sc.closeChan)
	if sc.conn != nil {
		return sc.conn.Close(websocket.StatusNormalClosure, "")
	}
	sc.logger.Info("Standalone WebSocket connection closed")
	return nil
}
