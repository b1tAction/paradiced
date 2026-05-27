package minigame

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/id"
)

func TestColyseusProviderCreateRoom(t *testing.T) {
	provider := NewColyseusProvider(ColyseusProviderConfig{
		PublicWSURL:   "ws://127.0.0.1:2567",
		NakamaRPCURL:  "http://nakama:7350/v2/rpc/minigame_result_callback",
		Secret:        "test_secret",
		NakamaMatchID: "match_abc",
	})

	playerIDs := []string{"player1", "player2", "player3"}
	conn, err := provider.CreateRoom(constants.MiniGameTypeDilemmaRace, playerIDs)
	if err != nil {
		t.Fatalf("CreateRoom failed: %v", err)
	}

	// Verify URL is the public WebSocket URL
	if conn.URL != "ws://127.0.0.1:2567" {
		t.Errorf("expected URL 'ws://127.0.0.1:2567', got '%s'", conn.URL)
	}

	// Verify RoomName matches game type
	if conn.RoomName != "dilemma_race" {
		t.Errorf("expected RoomName 'dilemma_race', got '%s'", conn.RoomName)
	}

	// Verify NakamaMatchID is set correctly
	if conn.NakamaMatchID != "match_abc" {
		t.Errorf("expected NakamaMatchID 'match_abc', got '%s'", conn.NakamaMatchID)
	}

	// Verify MiniGameInstanceID is not empty
	if conn.MiniGameInstanceID == "" {
		t.Error("expected MiniGameInstanceID to be non-empty")
	}

	// Verify CreatorPlayerID is the first player
	if conn.CreatorPlayerID != "player1" {
		t.Errorf("expected CreatorPlayerID 'player1', got '%s'", conn.CreatorPlayerID)
	}

	// Verify RoomID is empty (WebSocket mode - no REST room creation)
	if conn.RoomID != "" {
		t.Errorf("expected RoomID to be empty in WS mode, got '%s'", conn.RoomID)
	}

	// Verify PlayerTokens count matches player count
	if len(conn.PlayerTokens) != 3 {
		t.Errorf("expected 3 player tokens, got %d", len(conn.PlayerTokens))
	}

	// Verify each player has a valid token
	for _, pid := range playerIDs {
		token, ok := conn.PlayerTokens[pid]
		if !ok {
			t.Errorf("missing token for player %s", pid)
		}
		if !provider.VerifyToken(token, pid, "match_abc", conn.MiniGameInstanceID) {
			t.Errorf("token verification failed for player %s", pid)
		}
	}

	// Verify tokens are different per player
	token1 := conn.PlayerTokens["player1"]
	token2 := conn.PlayerTokens["player2"]
	if token1 == token2 {
		t.Error("different players should get different tokens")
	}
}

func TestColyseusProviderCreateRoomDifferentInstances(t *testing.T) {
	provider := NewColyseusProvider(ColyseusProviderConfig{
		PublicWSURL:   "ws://127.0.0.1:2567",
		Secret:        "test_secret",
		NakamaMatchID: "match_abc",
	})

	playerIDs := []string{"player1", "player2"}

	// Create two rooms - they should have different instance IDs and tokens
	conn1, _ := provider.CreateRoom(constants.MiniGameTypeDilemmaRace, playerIDs)
	conn2, _ := provider.CreateRoom(constants.MiniGameTypeDilemmaRace, playerIDs)

	if conn1.MiniGameInstanceID == conn2.MiniGameInstanceID {
		t.Error("different CreateRoom calls should produce different instance IDs")
	}

	// Same player should get different tokens for different instances
	token1P1 := conn1.PlayerTokens["player1"]
	token2P1 := conn2.PlayerTokens["player1"]
	if token1P1 == token2P1 {
		t.Error("same player in different instances should get different tokens")
	}
}

func TestColyseusProviderDestroyRoom(t *testing.T) {
	provider := NewColyseusProvider(ColyseusProviderConfig{
		PublicWSURL: "ws://127.0.0.1:2567",
		Secret:      "test_secret",
	})

	// DestroyRoom is now a no-op (rooms auto-dispose after result callback)
	err := provider.DestroyRoom("test_room_123")
	if err != nil {
		t.Fatalf("DestroyRoom should return nil (no-op), got: %v", err)
	}

	// Also works for non-existent room IDs
	err = provider.DestroyRoom("nonexistent_room")
	if err != nil {
		t.Fatalf("DestroyRoom should return nil for any room ID, got: %v", err)
	}
}

func TestColyseusProviderGetTimeout(t *testing.T) {
	provider := NewColyseusProvider(ColyseusProviderConfig{
		PublicWSURL: "ws://127.0.0.1:2567",
		Secret:      "test_secret",
	})

	// Online games rely on the Colyseus room to finish and report results.
	timeout := provider.GetTimeout(constants.MiniGameTypeDilemmaRace)
	if timeout != 0 {
		t.Errorf("expected disabled Nakama-side timeout for dilemma_race, got %v", timeout)
	}

	timeout = provider.GetTimeout(constants.MiniGameType("some_other_online"))
	if timeout != 0 {
		t.Errorf("expected disabled Nakama-side timeout by default, got %v", timeout)
	}
}

func TestColyseusProviderTokenGeneration(t *testing.T) {
	secret := "my_shared_secret"
	nakamaMatchID := "match_xyz"
	provider := NewColyseusProvider(ColyseusProviderConfig{
		PublicWSURL:   "ws://127.0.0.1:2567",
		Secret:        secret,
		NakamaMatchID: nakamaMatchID,
	})

	// Verify token format: hmac_sha256(secret, playerID + ":" + nakamaMatchID + ":" + instanceID)
	playerID := "player_uuid_1"
	instanceID := id.NewMiniGameInstanceID().UUID()
	token := provider.generateToken(playerID, nakamaMatchID, instanceID)

	// Verify it matches manual HMAC calculation
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(playerID + ":" + nakamaMatchID + ":" + instanceID))
	expected := hex.EncodeToString(mac.Sum(nil))

	if token != expected {
		t.Errorf("generated token doesn't match expected HMAC")
	}

	// Verify VerifyToken works with all parameters
	if !provider.VerifyToken(token, playerID, nakamaMatchID, instanceID) {
		t.Error("VerifyToken should return true for valid token")
	}

	// Verify wrong token fails
	if provider.VerifyToken("wrong_token", playerID, nakamaMatchID, instanceID) {
		t.Error("VerifyToken should return false for invalid token")
	}

	// Verify wrong player fails
	if provider.VerifyToken(token, "wrong_player", nakamaMatchID, instanceID) {
		t.Error("VerifyToken should return false for wrong player ID")
	}

	// Verify wrong nakamaMatchID fails
	if provider.VerifyToken(token, playerID, "wrong_match", instanceID) {
		t.Error("VerifyToken should return false for wrong nakama match ID")
	}

	// Verify wrong instanceID fails
	if provider.VerifyToken(token, playerID, nakamaMatchID, "wrong_instance") {
		t.Error("VerifyToken should return false for wrong instance ID")
	}

	// Verify tokens are different per player
	token2 := provider.generateToken("player_uuid_2", nakamaMatchID, instanceID)
	if token == token2 {
		t.Error("different players should get different tokens")
	}

	// Verify tokens are different per instanceID
	token3 := provider.generateToken(playerID, nakamaMatchID, "different_instance")
	if token == token3 {
		t.Error("same player with different instance ID should get different token")
	}
}
