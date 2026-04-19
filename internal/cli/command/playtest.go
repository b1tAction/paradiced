// Package command provides CLI commands for paradiced CLI.
package command

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/b1tAction/paradiced/internal/cli/nakama"
	"github.com/b1tAction/paradiced/internal/cli/scenario"
	"github.com/spf13/cobra"
)

var (
	// Playtest flags
	playersCount    int
	matchName       string
	maxTurns        int
	timeoutSec      int
	outputFile      string
	serverHTTP      string
	serverWS        string
	serverKey       string
	serverMode      string
	verbose         bool
)

// playtestCmd represents the playtest command.
var playtestCmd = &cobra.Command{
	Use:   "playtest",
	Short: "Run automated playtest",
	Long:  `Run automated playtest to verify backend playability`,
}

// playtestRunCmd represents the playtest run command.
var playtestRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run single automated match",
	Long:  `Run single automated match, supports 2-4 players`,
	RunE:  runPlaytest,
}

// playtestSoakCmd represents the playtest soak command.
var playtestSoakCmd = &cobra.Command{
	Use:   "soak",
	Short: "Run soak test (multiple rounds)",
	Long:  `Run soak test (multiple rounds) to verify stability`,
	RunE:  runSoak,
}

func init() {
	// Add playtest command to root
	AddCommand(playtestCmd)

	// Add subcommands
	playtestCmd.AddCommand(playtestRunCmd)
	playtestCmd.AddCommand(playtestSoakCmd)

	// playtest run flags
	playtestRunCmd.Flags().IntVar(&playersCount, "players", 4, "Number of players (1-4)")
	playtestRunCmd.Flags().StringVar(&matchName, "match-name", "paradiced_match", "Match name")
	playtestRunCmd.Flags().IntVar(&maxTurns, "max-turns", 50, "Maximum turns")
	playtestRunCmd.Flags().IntVar(&timeoutSec, "timeout", 180, "Timeout in seconds")
	playtestRunCmd.Flags().StringVar(&outputFile, "output", "", "Output JSON report path")
	playtestRunCmd.Flags().StringVar(&serverHTTP, "server-http", "http://127.0.0.1:7350", "Nakama HTTP address")
	playtestRunCmd.Flags().StringVar(&serverWS, "server-ws", "ws://127.0.0.1:7350/ws", "Nakama WebSocket address")
	playtestRunCmd.Flags().StringVar(&serverKey, "server-key", "defaultkey", "Nakama server key")
	playtestRunCmd.Flags().StringVar(&serverMode, "mode", "nakama", "Server mode: nakama or standalone")
	playtestRunCmd.Flags().BoolVar(&verbose, "verbose", false, "Enable verbose logging")

	// playtest soak flags
	playtestSoakCmd.Flags().IntVar(&playersCount, "players", 2, "Number of players (1-4)")
	playtestSoakCmd.Flags().IntVar(&maxTurns, "rounds", 20, "Number of rounds to test")
	playtestSoakCmd.Flags().StringVar(&serverHTTP, "server-http", "http://127.0.0.1:7350", "Nakama HTTP address")
	playtestSoakCmd.Flags().StringVar(&serverWS, "server-ws", "ws://127.0.0.1:7350/ws", "Nakama WebSocket address")
	playtestSoakCmd.Flags().StringVar(&serverKey, "server-key", "defaultkey", "Nakama server key")
	playtestSoakCmd.Flags().StringVar(&serverMode, "mode", "nakama", "Server mode: nakama or standalone")
	playtestSoakCmd.Flags().BoolVar(&verbose, "verbose", false, "Enable verbose logging")

	// Flags are already set with defaults, no need to mark as required
}

func runPlaytest(cmd *cobra.Command, args []string) error {
	// Validate players count
	if playersCount < 1 || playersCount > 4 {
		return fmt.Errorf("player count must be between 1 and 4")
	}

	// Validate mode
	if serverMode != "nakama" && serverMode != "standalone" {
		return fmt.Errorf("invalid mode: %s, must be 'nakama' or 'standalone'", serverMode)
	}

	// Create logger
	logger := nakama.NewLogger(verbose)

	// Create client based on mode
	var client nakama.IClient
	var err error

	if serverMode == "standalone" {
		// Create standalone WebSocket client
		client, err = nakama.NewStandaloneClient(nakama.StandaloneClientConfig{
			ServerWS: serverWS,
			Verbose:  verbose,
		})
	} else {
		// Create Nakama client
		client, err = nakama.NewClient(nakama.ClientConfig{
			ServerHTTP: serverHTTP,
			ServerWS:   serverWS,
			ServerKey:  serverKey,
			Verbose:    verbose,
		})
	}
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	logger.Info("Starting playtest", "players", playersCount, "max_turns", maxTurns, "mode", serverMode)

	// Create scenario config
	config := scenario.ScenarioConfig{
		PlayersCount: playersCount,
		MatchName:    matchName,
		MaxTurns:     maxTurns,
		TimeoutSec:   timeoutSec,
		Mode:         serverMode,
	}

	// Run autoplay scenario
	result, err := scenario.RunAutoPlay(cmd.Context(), client, config, logger)
	if err != nil {
		logger.Error("Playtest failed", "error", err)
		return err
	}

	// Print summary
	printSummary(result)

	// Write JSON report if output file specified
	if outputFile != "" {
		if err := writeJSONReport(result, outputFile); err != nil {
			logger.Warn("Failed to write JSON report", "error", err)
		} else {
			logger.Info("JSON report written", "file", outputFile)
		}
	}

	if result.Success {
		logger.Info("Playtest completed successfully")
		return nil
	}

	return fmt.Errorf("playtest failed: %s", result.FailureReason)
}

func runSoak(cmd *cobra.Command, args []string) error {
	// Validate players count
	if playersCount < 1 || playersCount > 4 {
		return fmt.Errorf("player count must be between 1 and 4")
	}

	// Validate mode
	if serverMode != "nakama" && serverMode != "standalone" {
		return fmt.Errorf("invalid mode: %s, must be 'nakama' or 'standalone'", serverMode)
	}

	// Create logger
	logger := nakama.NewLogger(verbose)

	// Create client based on mode
	var client nakama.IClient
	var err error

	if serverMode == "standalone" {
		// Create standalone WebSocket client
		client, err = nakama.NewStandaloneClient(nakama.StandaloneClientConfig{
			ServerWS: serverWS,
			Verbose:  verbose,
		})
	} else {
		// Create Nakama client
		client, err = nakama.NewClient(nakama.ClientConfig{
			ServerHTTP: serverHTTP,
			ServerWS:   serverWS,
			ServerKey:  serverKey,
			Verbose:    verbose,
		})
	}
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	logger.Info("Starting soak test", "rounds", maxTurns, "players", playersCount, "mode", serverMode)

	// Run multiple rounds
	successCount := 0
	failCount := 0

	for i := 0; i < maxTurns; i++ {
		logger.Info(fmt.Sprintf("Round %d/%d", i+1, maxTurns))

		config := scenario.ScenarioConfig{
			PlayersCount: playersCount,
			MatchName:    "paradiced_match",
			MaxTurns:     50,
			TimeoutSec:   180,
			Mode:         serverMode,
		}

		result, err := scenario.RunAutoPlay(cmd.Context(), client, config, logger)
		if err != nil || !result.Success {
			failCount++
			logger.Warn(fmt.Sprintf("Round %d failed", i+1), "error", err)
		} else {
			successCount++
			logger.Info(fmt.Sprintf("Round %d succeeded", i+1))
		}
	}

	// Print soak summary
	total := successCount + failCount
	successRate := float64(successCount) / float64(total) * 100

	logger.Info("========== Soak Test Results ==========")
	logger.Info(fmt.Sprintf("Total rounds: %d", total))
	logger.Info(fmt.Sprintf("Success: %d (%.1f%%)", successCount, successRate))
	logger.Info(fmt.Sprintf("Failure: %d (%.1f%%)", failCount, 100-successRate))
	logger.Info("========================================")

	if successRate < 90 {
		return fmt.Errorf("success rate below 90%% (%.1f%%)", successRate)
	}

	return nil
}

func printSummary(result scenario.Result) {
	fmt.Println("\n========== Match Results ==========")
	fmt.Printf("Status: %s\n", map[bool]string{true: "Success", false: "Failed"}[result.Success])
	if !result.Success {
		fmt.Printf("Failure Reason: %s\n", result.FailureReason)
	}
	fmt.Printf("Duration: %.2f seconds\n", result.Duration.Seconds())
	fmt.Printf("Messages Received: %d\n", result.MessagesReceived)
	fmt.Printf("Turns Completed: %d\n", result.TurnsCompleted)
	fmt.Println("====================================")
}

func writeJSONReport(result scenario.Result, outputPath string) error {
	// Create directory if not exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(outputPath, data, 0644)
}
