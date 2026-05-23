// Package minigame provides mini-game type selection, rank calculation, and online provider.
package minigame

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/id"
	pkgnet "github.com/b1tAction/paradiced/pkg/net"
	"github.com/b1tAction/paradiced/pkg/protocol"
)

// ColyseusProvider implements protocol.OnlineMiniGameProvider as a credential generator.
// In WebSocket/matchmaker mode, it no longer creates rooms via REST API.
// Instead, it generates signed session credentials (MiniGameConn) that clients
// use to joinOrCreate Colyseus rooms directly via WebSocket.
type ColyseusProvider struct {
	// publicWSURL is the browser-accessible Colyseus WebSocket URL
	// (e.g. "ws://127.0.0.1:2567" or "wss://example.com/game/paradice/minigame").
	publicWSURL string

	// nakamaMatchID is the Nakama runtime match ID, used for:
	// (1) token generation (player_id + nakama_match_id + minigame_instance_id)
	// (2) result callback routing (Colyseus sends rankings back to this match)
	nakamaMatchID string

	// secret is the shared secret for HMAC token generation and RPC callback validation.
	secret string
}

// ColyseusProviderConfig holds configuration for creating a ColyseusProvider.
type ColyseusProviderConfig struct {
	PublicWSURL   string // Browser-accessible WebSocket URL (e.g. "ws://127.0.0.1:2567")
	NakamaRPCURL  string // Nakama RPC endpoint URL (unused in WS mode, kept for compat)
	Secret        string // Shared secret for HMAC and RPC auth
	NakamaMatchID string // Nakama runtime match ID for result callback routing
}

// NewColyseusProvider creates a new ColyseusProvider with the given configuration.
func NewColyseusProvider(cfg ColyseusProviderConfig) *ColyseusProvider {
	return &ColyseusProvider{
		publicWSURL:   cfg.PublicWSURL,
		nakamaMatchID: cfg.NakamaMatchID,
		secret:        cfg.Secret,
	}
}

// CreateRoom generates a session connection credential for clients to join
// a Colyseus room via WebSocket/matchmaker (joinOrCreate).
// No REST API call is made; the actual room is created by the first client
// to call joinOrCreate on the Colyseus server.
func (p *ColyseusProvider) CreateRoom(gameType constants.MiniGameType, playerIDs []string) (*pkgnet.MiniGameConn, error) {
	instanceID := id.NewMiniGameInstanceID().UUID()

	// Generate per-player HMAC tokens using playerID + nakamaMatchID + instanceID
	playerTokens := make(map[string]string)
	for _, pid := range playerIDs {
		playerTokens[pid] = p.generateToken(pid, p.nakamaMatchID, instanceID)
	}

	return &pkgnet.MiniGameConn{
		URL:                p.publicWSURL,
		RoomName:           string(gameType),
		NakamaMatchID:      p.nakamaMatchID,
		MiniGameInstanceID: instanceID,
		CreatorPlayerID:    playerIDs[0],
		PlayerTokens:       playerTokens,
	}, nil
}

// DestroyRoom is a no-op in WebSocket/matchmaker mode.
// Colyseus rooms auto-dispose after sending results via RPC callback.
func (p *ColyseusProvider) DestroyRoom(roomID string) error {
	return nil // no-op: room auto-disposes after result callback
}

// GetTimeout returns the maximum wait duration for a mini-game type.
// Default: 60 seconds for all online games.
func (p *ColyseusProvider) GetTimeout(gameType constants.MiniGameType) time.Duration {
	switch gameType {
	case constants.MiniGameTypeDilemmaRace:
		return 90 * time.Second // Dilemma race needs more time (up to 20 rounds)
	default:
		return 60 * time.Second
	}
}

// generateToken creates an HMAC-SHA256 token for a player.
// The token is derived from playerID + nakamaMatchID + instanceID + secret,
// ensuring each player gets a unique, verifiable token for Colyseus onAuth.
func (p *ColyseusProvider) generateToken(playerID, nakamaMatchID, instanceID string) string {
	mac := hmac.New(sha256.New, []byte(p.secret))
	mac.Write([]byte(playerID + ":" + nakamaMatchID + ":" + instanceID))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyToken checks if a token is valid for the given playerID, nakamaMatchID, and instanceID.
// Used by Colyseus onAuth to validate player join requests.
func (p *ColyseusProvider) VerifyToken(token, playerID, nakamaMatchID, instanceID string) bool {
	expected := p.generateToken(playerID, nakamaMatchID, instanceID)
	return hmac.Equal([]byte(token), []byte(expected))
}

// Compile-time interface check.
var _ protocol.OnlineMiniGameProvider = (*ColyseusProvider)(nil)