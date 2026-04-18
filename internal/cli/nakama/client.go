// Package nakama provides Nakama client wrapper for CLI.
package nakama

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ascii8/nakama-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

	c.logger.Debug("认证成功", "user_id", account.User.Id)
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
}

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
		if msg.Presence != nil && sc.session != nil && msg.Presence.UserId == sc.session.UserID {
			return
		}

		sc.logger.Debug("收到匹配消息", "op_code", msg.OpCode)

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
	sc.logger.Info("WebSocket 连接成功")

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
	sc.logger.Info("创建匹配成功", "match_id", sc.matchID)
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
		sc.logger.Debug("使用 Token 加入匹配", "token_len", len(matchIDOrToken))
		match, err = sc.conn.MatchJoinToken(ctx, matchIDOrToken, nil)
	} else {
		// Use as match ID
		sc.logger.Debug("使用 Match ID 加入匹配", "match_id", matchIDOrToken)
		match, err = sc.conn.MatchJoin(ctx, matchIDOrToken, nil)
	}

	if err != nil {
		return err
	}

	sc.matchID = match.MatchId
	sc.logger.Info("加入匹配成功", "match_id", sc.matchID)
	return nil
}

// SendMessage sends a message to the match.
func (sc *SocketClient) SendMessage(ctx context.Context, opCode int64, data any) error {
	// Marshal data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return sc.conn.MatchDataSend(ctx, sc.matchID, opCode, jsonData, true, nil)
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
	sc.logger.Info("WebSocket 连接已关闭")
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
