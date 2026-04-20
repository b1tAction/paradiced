// Package command provides CLI commands for paradiced CLI.
package command

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/b1tAction/paradiced/internal/cli/nakama"
	"github.com/b1tAction/paradiced/internal/cli/player"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/spf13/cobra"
)

// hostCmd is the command for creating a game room.
var hostCmd = &cobra.Command{
	Use:   "host",
	Short: "Create a game room and wait for players",
	Long: `Create a new game room (authoritative match) and wait for other players to join.
The game will start automatically when the room is full (4 players).

Examples:
  # Create a room with default settings
  pdcli host --server-http=http://127.0.0.1:7350 --user-id=myname

  # Create a room with custom name and faction
  pdcli host --match-name=myroom --faction=zhu_que --verbose`,
	RunE: runHost,
}

var (
	hostServerHTTP   string
	hostServerWS     string
	hostServerKey    string
	hostUserID       string
	hostMatchName    string
	hostFaction      string
	hostMaxPlayers   int
	hostVerbose      bool
	hostTimeout      int
)

func init() {
	hostCmd.Flags().StringVar(&hostServerHTTP, "server-http", "http://127.0.0.1:7350", "Nakama HTTP server URL")
	hostCmd.Flags().StringVar(&hostServerWS, "server-ws", "ws://127.0.0.1:7350/ws", "Nakama WebSocket server URL")
	hostCmd.Flags().StringVar(&hostServerKey, "server-key", "defaultkey", "Nakama server key")
	hostCmd.Flags().StringVar(&hostUserID, "user-id", "", "User ID for authentication (required)")
	hostCmd.Flags().StringVar(&hostMatchName, "match-name", "paradiced_match", "Match name for room")
	hostCmd.Flags().StringVar(&hostFaction, "faction", "qing_long", "Faction choice (qing_long, zhu_que, bai_hu, xuan_wu)")
	hostCmd.Flags().IntVar(&hostMaxPlayers, "max-players", 4, "Maximum players for the room")
	hostCmd.Flags().BoolVar(&hostVerbose, "verbose", false, "Enable verbose logging")
	hostCmd.Flags().IntVar(&hostTimeout, "timeout", 600, "Game timeout in seconds (including waiting)")

	hostCmd.MarkFlagRequired("user-id")
}

func runHost(cmd *cobra.Command, args []string) error {
	// Validate faction
	faction := constants.ParseFaction(hostFaction)
	if !faction.IsValid() {
		return fmt.Errorf("invalid faction: %s (valid: qing_long, zhu_que, bai_hu, xuan_wu)", hostFaction)
	}

	// Validate max players
	if hostMaxPlayers < 2 || hostMaxPlayers > 4 {
		return fmt.Errorf("max-players must be between 2 and 4")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(hostTimeout)*time.Second)
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case <-sigChan:
			fmt.Println("\nReceived exit signal, shutting down...")
			cancel()
		case <-ctx.Done():
		}
	}()

	// Create logger
	logger := nakama.NewLogger(hostVerbose)

	// Create Nakama client
	clientConfig := nakama.ClientConfig{
		ServerHTTP: hostServerHTTP,
		ServerWS:   hostServerWS,
		ServerKey:  hostServerKey,
		Verbose:    hostVerbose,
	}

	client, err := nakama.NewClient(clientConfig)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	// Authenticate
	session, err := client.Authenticate(ctx, hostUserID)
	if err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}

	logger.Info("Authentication successful", "user_id", session.UserID)

	// Create socket client
	socketClient, err := client.CreateSocketClient()
	if err != nil {
		return fmt.Errorf("failed to create socket client: %w", err)
	}
	defer socketClient.Close()

	// Connect WebSocket
	if err := socketClient.Connect(ctx, session); err != nil {
		return fmt.Errorf("failed to connect WebSocket: %w", err)
	}

	logger.Info("WebSocket connected")

	// Call RPC to create authoritative room on server
	// IMPORTANT: socket.CreateMatch creates a relayed match (not authoritative)
	// We must use server-side RPC to create an authoritative match via nk.MatchCreate
	matchID, err := socketClient.Rpc(ctx, "create_authoritative_room", "")
	if err != nil {
		return fmt.Errorf("failed to create authoritative room via RPC: %w", err)
	}

	logger.Info("Authoritative room created", "match_id", matchID)

	// Host must explicitly join the match (MatchJoin will be triggered on server)
	if err := socketClient.JoinMatch(ctx, matchID); err != nil {
		return fmt.Errorf("host failed to join match: %w", err)
	}

	logger.Info("Host joined match", "match_id", matchID)

	// Create UI adapter
	uiAdapter := player.NewCLIUIAdapter(session.UserID, faction, hostVerbose)
	uiAdapter.OnMatchCreated(matchID)

	// Create interactive player
	interactivePlayer := player.NewInteractivePlayer(socketClient, session.UserID, uiAdapter, logger)

	uiAdapter.OnWaiting("Waiting for players to join...", 1, hostMaxPlayers)

	// Start message listener
	go interactivePlayer.Listen(ctx)

	// Wait for game to start or timeout
	// NOTE: stdin is handled by OnWaitingSync in interactive.go, not here
	// Do not add stdin reading here to avoid conflict with bufio.Scanner in CLIUIAdapter
	fmt.Println("Press Ctrl+C to exit, or wait for players to join...")

	// Monitor for game start
	gameStarted := false
	checkInterval := time.NewTicker(500 * time.Millisecond)
	defer checkInterval.Stop()

	// Check if game has started (state changed from WaitingForHost)
	// The game start is handled by WaitingSync -> OnWaitingSync -> sendStartGame
	for !gameStarted {
		select {
		case <-ctx.Done():
			if !gameStarted {
				fmt.Println("\nWait timeout")
			}
			return ctx.Err()

		case <-interactivePlayer.GameOverChan():
			gameStarted = true
			fmt.Println("\nGame over!")

		case <-checkInterval.C:
			state := interactivePlayer.GlobalState()
			// Game started when state transitions from WaitingForHost to RoundMiniGame
			if state != "" && state != "match_init" && state != "waiting_for_host" {
				if !gameStarted {
					gameStarted = true
					fmt.Println("\nGame has started!")
				}
			}
		}
	}

	// Game started, wait for game over
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-interactivePlayer.GameOverChan():
		fmt.Println("\nGame over!")
	}

	return nil
}