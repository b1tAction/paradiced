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
	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/spf13/cobra"
)

// playCmd is the command for starting an interactive game.
var playCmd = &cobra.Command{
	Use:   "play",
	Short: "Start interactive game session",
	Long: `Start an interactive game session where you can create or join a room
and play the game with real-time user input.

Examples:
  # Connect to Nakama server and wait for commands
  pdcli play --server-http=http://127.0.0.1:7350 --user-id=myname

  # Specify faction
  pdcli play --faction=zhu_que --verbose`,
	RunE: runPlay,
}

var (
	playServerHTTP   string
	playServerWS     string
	playServerKey    string
	playUserID       string
	playFaction      string
	playVerbose      bool
	playTimeout      int
)

func init() {
	playCmd.Flags().StringVar(&playServerHTTP, "server-http", "http://127.0.0.1:7350", "Nakama HTTP server URL")
	playCmd.Flags().StringVar(&playServerWS, "server-ws", "ws://127.0.0.1:7350/ws", "Nakama WebSocket server URL")
	playCmd.Flags().StringVar(&playServerKey, "server-key", "defaultkey", "Nakama server key")
	playCmd.Flags().StringVar(&playUserID, "user-id", "", "User ID for authentication (required)")
	playCmd.Flags().StringVar(&playFaction, "faction", "qing_long", "Faction choice (qing_long, zhu_que, bai_hu, xuan_wu)")
	playCmd.Flags().BoolVar(&playVerbose, "verbose", false, "Enable verbose logging")
	playCmd.Flags().IntVar(&playTimeout, "timeout", 300, "Game timeout in seconds")

	playCmd.MarkFlagRequired("user-id")
}

func runPlay(cmd *cobra.Command, args []string) error {
	// Validate faction
	faction := constants.ParseFaction(playFaction)
	if !faction.IsValid() {
		return fmt.Errorf("invalid faction: %s (valid: qing_long, zhu_que, bai_hu, xuan_wu)", playFaction)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(playTimeout)*time.Second)
	defer cancel()

	// Setup signal handling for graceful shutdown
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
	logger := nakama.NewLogger(playVerbose)

	// Create Nakama client
	clientConfig := nakama.ClientConfig{
		ServerHTTP: playServerHTTP,
		ServerWS:   playServerWS,
		ServerKey:  playServerKey,
		Verbose:    playVerbose,
	}

	client, err := nakama.NewClient(clientConfig)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	// Authenticate
	session, err := client.Authenticate(ctx, playUserID)
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

	// Display welcome message
	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("  ParaDiced CLI - Interactive Mode")
	fmt.Println("========================================")
	fmt.Printf("  User: %s\n", playUserID)
	fmt.Printf("  Faction: %s\n", string(faction))
	fmt.Println("========================================")
	fmt.Println()
	fmt.Println("Available commands:")
	fmt.Println("  - Use 'pdcli host' to create a room")
	fmt.Println("  - Use 'pdcli join' to join a room")
	fmt.Println()
	fmt.Println("Use host or join command to start the game")
	fmt.Println()

	return nil
}