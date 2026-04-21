// Package command provides CLI commands for paradiced CLI.
package command

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/b1tAction/paradiced/internal/cli/nakama"
	"github.com/b1tAction/paradiced/internal/cli/player"
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/spf13/cobra"
)

// joinCmd is the command for joining an existing game room.
var joinCmd = &cobra.Command{
	Use:   "join",
	Short: "Join an existing game room",
	Long: `Join an existing game room by match ID.
You can get the match ID from the host player who created the room.

Examples:
	# Join a room with display name (auto-generate user-id)
	pdcli join --match-id=7c9b4d2a-xxxx.xxxx --name=alice

  # Join with specific faction
  pdcli join --match-id=<match-id> --faction=bai_hu --verbose`,
	RunE: runJoin,
}

var (
	joinServerHTTP string
	joinServerWS   string
	joinServerKey  string
	joinName       string
	joinMatchID    string
	joinFaction    string
	joinVerbose    bool
	joinTimeout    int
)

func init() {
	joinCmd.Flags().StringVar(&joinServerHTTP, "server-http", "http://127.0.0.1:7350", "Nakama HTTP server URL")
	joinCmd.Flags().StringVar(&joinServerWS, "server-ws", "ws://127.0.0.1:7350/ws", "Nakama WebSocket server URL")
	joinCmd.Flags().StringVar(&joinServerKey, "server-key", "defaultkey", "Nakama server key")
	joinCmd.Flags().StringVar(&joinName, "name", "", "Display name (required), used to auto-generate user-id")
	joinCmd.Flags().StringVar(&joinMatchID, "match-id", "", "Match ID to join (required)")
	joinCmd.Flags().StringVar(&joinFaction, "faction", "qing_long", "Faction choice (qing_long, zhu_que, bai_hu, xuan_wu)")
	joinCmd.Flags().BoolVar(&joinVerbose, "verbose", false, "Enable verbose logging")
	joinCmd.Flags().IntVar(&joinTimeout, "timeout", 600, "Game timeout in seconds")

	joinCmd.MarkFlagRequired("name")
	joinCmd.MarkFlagRequired("match-id")
}

func runJoin(cmd *cobra.Command, args []string) error {
	// Validate faction
	faction := constants.ParseFaction(joinFaction)
	if !faction.IsValid() {
		return fmt.Errorf("invalid faction: %s (valid: qing_long, zhu_que, bai_hu, xuan_wu)", joinFaction)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(joinTimeout)*time.Second)
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
	logger := nakama.NewLogger(joinVerbose)

	resolvedUserID, err := resolveUserID(joinName)
	if err != nil {
		return err
	}

	// Create Nakama client
	clientConfig := nakama.ClientConfig{
		ServerHTTP: joinServerHTTP,
		ServerWS:   joinServerWS,
		ServerKey:  joinServerKey,
		Verbose:    joinVerbose,
	}

	client, err := nakama.NewClient(clientConfig)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	// Authenticate
	session, err := client.Authenticate(ctx, resolvedUserID)
	if err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}

	if joinVerbose {
		fmt.Printf("Using user-id: %s\n", session.UserID)
	}

	logger.Debug("Authentication successful", "user_id", session.UserID)

	// Create socket client
	socketClient, err := client.CreateSocketClient()
	if err != nil {
		return fmt.Errorf("failed to create socket client: %w", err)
	}
	defer func() {
		leaveCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = socketClient.LeaveMatch(leaveCtx)
	}()
	defer socketClient.Close()

	// Connect WebSocket
	if err := socketClient.Connect(ctx, session); err != nil {
		return fmt.Errorf("failed to connect WebSocket: %w", err)
	}

	logger.Debug("WebSocket connected")

	// Join match with metadata (faction and display_name)
	metadata := map[string]string{
		"faction":      joinFaction,
		"display_name": joinName,
	}
	if err := socketClient.JoinMatch(ctx, joinMatchID, metadata); err != nil {
		return fmt.Errorf("failed to join match: %w", err)
	}

	// Create UI adapter
	uiAdapter := player.NewCLIUIAdapter(session.UserID, faction, joinVerbose)
	uiAdapter.OnConnected(joinMatchID)

	// Create interactive player
	interactivePlayer := player.NewInteractivePlayer(socketClient, session.UserID, uiAdapter, logger)

	// Start message listener
	go interactivePlayer.Listen(ctx)

	// Display waiting status
	fmt.Println("Joined room, waiting for game to start...")
	fmt.Println("Press Ctrl+C to exit")
	fmt.Println()

	// Monitor for game start
	gameStarted := false
	checkInterval := time.NewTicker(500 * time.Millisecond)
	defer checkInterval.Stop()

	reader := bufio.NewReader(os.Stdin)

	for !gameStarted {
		select {
		case <-ctx.Done():
			if !gameStarted {
				fmt.Println("\nWait timeout")
			}
			if ctx.Err() == context.Canceled {
				return nil
			}
			return ctx.Err()

		case <-interactivePlayer.GameOverChan():
			gameStarted = true
			fmt.Println("\nGame over!")

		case <-checkInterval.C:
			state := interactivePlayer.GlobalState()
			if state != "" && state != "match_init" {
				if !gameStarted {
					gameStarted = true
					fmt.Println("\nGame has started!")
				}
			}

		default:
			// Check for user input (non-blocking)
			// This is a simplified approach - in production would use proper async input
			if reader.Buffered() > 0 {
				if input, err := reader.ReadString('\n'); err == nil {
					if handleJoinCommand(strings.TrimSpace(input), interactivePlayer) {
						cancel()
						return nil
					}
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

// handleJoinCommand handles user commands during game.
func handleJoinCommand(input string, interactivePlayer *player.InteractivePlayer) bool {
	switch strings.ToLower(input) {
	case "status", "s":
		interactivePlayer.DisplayDetailedStatus()
		return false
	case "help", "h":
		fmt.Println("\nAvailable commands:")
		fmt.Println("  status/s - View current status")
		fmt.Println("  help/h   - Show help")
		fmt.Println("  quit/q   - Exit game")
		return false
	case "quit", "q":
		fmt.Println("\nExiting game...")
		return true
	default:
		if input != "" {
			fmt.Printf("\nUnknown command: %s (type 'help' for available commands)\n", input)
		}
		return false
	}
}
