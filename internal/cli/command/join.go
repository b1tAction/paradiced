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
  # Join a room with match ID
  pdcli join --match-id=7c9b4d2a-xxxx.xxxx --user-id=myname

  # Join with specific faction
  pdcli join --match-id=<match-id> --faction=bai_hu --verbose`,
	RunE: runJoin,
}

var (
	joinServerHTTP   string
	joinServerWS     string
	joinServerKey    string
	joinUserID       string
	joinMatchID      string
	joinFaction      string
	joinVerbose      bool
	joinTimeout      int
)

func init() {
	joinCmd.Flags().StringVar(&joinServerHTTP, "server-http", "http://127.0.0.1:7350", "Nakama HTTP server URL")
	joinCmd.Flags().StringVar(&joinServerWS, "server-ws", "ws://127.0.0.1:7350/ws", "Nakama WebSocket server URL")
	joinCmd.Flags().StringVar(&joinServerKey, "server-key", "defaultkey", "Nakama server key")
	joinCmd.Flags().StringVar(&joinUserID, "user-id", "", "User ID for authentication (required)")
	joinCmd.Flags().StringVar(&joinMatchID, "match-id", "", "Match ID to join (required)")
	joinCmd.Flags().StringVar(&joinFaction, "faction", "qing_long", "Faction choice (qing_long, zhu_que, bai_hu, xuan_wu)")
	joinCmd.Flags().BoolVar(&joinVerbose, "verbose", false, "Enable verbose logging")
	joinCmd.Flags().IntVar(&joinTimeout, "timeout", 600, "Game timeout in seconds")

	joinCmd.MarkFlagRequired("user-id")
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
	session, err := client.Authenticate(ctx, joinUserID)
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

	// Join match
	if err := socketClient.JoinMatch(ctx, joinMatchID); err != nil {
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
					handleJoinCommand(strings.TrimSpace(input), interactivePlayer)
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
func handleJoinCommand(input string, interactivePlayer *player.InteractivePlayer) {
	switch strings.ToLower(input) {
	case "status", "s":
		interactivePlayer.DisplayDetailedStatus()
	case "help", "h":
		fmt.Println("\nAvailable commands:")
		fmt.Println("  status/s - View current status")
		fmt.Println("  help/h   - Show help")
		fmt.Println("  quit/q   - Exit game")
	case "quit", "q":
		fmt.Println("\nExiting game...")
		os.Exit(0)
	default:
		if input != "" {
			fmt.Printf("\nUnknown command: %s (type 'help' for available commands)\n", input)
		}
	}
}