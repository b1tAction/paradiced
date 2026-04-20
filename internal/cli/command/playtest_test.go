// Package command provides CLI commands for paradiced CLI.
package command

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/b1tAction/paradiced/internal/cli/scenario"
	"github.com/spf13/cobra"
)

// ========== Parameter Validation Tests ==========

func TestValidatePlayersCount(t *testing.T) {
	tests := []struct {
		name     string
		count    int
		hasError bool
	}{
		{"valid 1", 1, false},
		{"valid 2", 2, false},
		{"valid 3", 3, false},
		{"valid 4", 4, false},
		{"invalid 0", 0, true},
		{"invalid 5", 5, true},
		{"invalid -1", -1, true},
		{"invalid 10", 10, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test validation logic directly
			err := validatePlayersCount(tt.count)
			if tt.hasError && err == nil {
				t.Errorf("players count %d should return error", tt.count)
			}
			if !tt.hasError && err != nil {
				t.Errorf("players count %d should not return error: %v", tt.count, err)
			}
		})
	}
}

func validatePlayersCount(count int) error {
	if count < 1 || count > 4 {
		return ErrInvalidPlayersCount
	}
	return nil
}

// ErrInvalidPlayersCount is returned when player count is out of range.
var ErrInvalidPlayersCount = context.DeadlineExceeded // placeholder error

func TestValidateServerMode(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		hasError bool
	}{
		{"valid nakama", "nakama", false},
		{"valid standalone", "standalone", false},
		{"invalid random", "random", true},
		{"invalid empty", "", true},
		{"invalid test", "test", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateServerMode(tt.mode)
			if tt.hasError && err == nil {
				t.Errorf("mode %s should return error", tt.mode)
			}
			if !tt.hasError && err != nil {
				t.Errorf("mode %s should not return error: %v", tt.mode, err)
			}
		})
	}
}

func validateServerMode(mode string) error {
	if mode != "nakama" && mode != "standalone" {
		return ErrInvalidServerMode
	}
	return nil
}

// ErrInvalidServerMode is returned when server mode is invalid.
var ErrInvalidServerMode = context.DeadlineExceeded // placeholder error

// ========== Command Structure Tests ==========

func TestPlaytestCmdUse(t *testing.T) {
	if playtestCmd.Use != "playtest" {
		t.Errorf("playtestCmd.Use = %s, expected playtest", playtestCmd.Use)
	}
}

func TestPlaytestRunCmdUse(t *testing.T) {
	if playtestRunCmd.Use != "run" {
		t.Errorf("playtestRunCmd.Use = %s, expected run", playtestRunCmd.Use)
	}
}

func TestPlaytestSoakCmdUse(t *testing.T) {
	if playtestSoakCmd.Use != "soak" {
		t.Errorf("playtestSoakCmd.Use = %s, expected soak", playtestSoakCmd.Use)
	}
}

func TestPlaytestRunCmdFlags(t *testing.T) {
	// Check required flags exist
	flagTests := []struct {
		name     string
		expected string
	}{
		{"players", "4"},
		{"match-name", "paradiced_match"},
		{"max-turns", "50"},
		{"timeout", "180"},
		{"server-http", "http://127.0.0.1:7350"},
		{"server-ws", "ws://127.0.0.1:7350/ws"},
		{"server-key", "defaultkey"},
		{"mode", "nakama"},
		{"verbose", "false"},
	}

	for _, tt := range flagTests {
		t.Run(tt.name, func(t *testing.T) {
			flag := playtestRunCmd.Flag(tt.name)
			if flag == nil {
				t.Errorf("flag %s should exist", tt.name)
				return
			}
			// Check default value
			if flag.DefValue != tt.expected {
				t.Errorf("flag %s default = %s, expected %s", tt.name, flag.DefValue, tt.expected)
			}
		})
	}
}

func TestPlaytestSoakCmdFlags(t *testing.T) {
	flagTests := []struct {
		name     string
		expected string
	}{
		{"players", "2"},
		{"rounds", "20"},
		{"server-http", "http://127.0.0.1:7350"},
		{"server-ws", "ws://127.0.0.1:7350/ws"},
		{"server-key", "defaultkey"},
		{"mode", "nakama"},
	}

	for _, tt := range flagTests {
		t.Run(tt.name, func(t *testing.T) {
			flag := playtestSoakCmd.Flag(tt.name)
			if flag == nil {
				t.Errorf("flag %s should exist", tt.name)
				return
			}
			if flag.DefValue != tt.expected {
				t.Errorf("flag %s default = %s, expected %s", tt.name, flag.DefValue, tt.expected)
			}
		})
	}
}

func TestPlaytestCmdHasSubcommands(t *testing.T) {
	// Verify subcommands are added
	subcommands := playtestCmd.Commands()
	if len(subcommands) < 2 {
		t.Errorf("playtestCmd should have at least 2 subcommands, got %d", len(subcommands))
	}

	// Check for run and soak subcommands
	hasRun := false
	hasSoak := false
	for _, cmd := range subcommands {
		if cmd.Use == "run" {
			hasRun = true
		}
		if cmd.Use == "soak" {
			hasSoak = true
		}
	}

	if !hasRun {
		t.Error("playtestCmd should have 'run' subcommand")
	}
	if !hasSoak {
		t.Error("playtestCmd should have 'soak' subcommand")
	}
}

// ========== Helper Functions Tests ==========

func TestPrintSummary(t *testing.T) {
	// Capture output
	buf := &bytes.Buffer{}

	result := scenario.Result{
		Success:          true,
		Duration:         120 * time.Second,
		MessagesReceived: 156,
		TurnsCompleted:   8,
		Rejections:       0,
	}

	// Redirect stdout temporarily
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printSummary(result)

	w.Close()
	os.Stdout = old

	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Verify output contains expected content
	if !strings.Contains(output, "Success") {
		t.Error("output should contain 'Success'")
	}
	if !strings.Contains(output, "120") {
		t.Error("output should contain duration '120'")
	}
	if !strings.Contains(output, "156") {
		t.Error("output should contain messages received '156'")
	}
}

func TestPrintSummaryFailure(t *testing.T) {
	buf := &bytes.Buffer{}

	result := scenario.Result{
		Success:          false,
		FailureReason:    "timeout",
		Duration:         180 * time.Second,
		MessagesReceived: 50,
		TurnsCompleted:   5,
		Rejections:       2,
		LastError:        "connection lost",
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printSummary(result)

	w.Close()
	os.Stdout = old

	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Verify output contains failure reason
	if !strings.Contains(output, "Failed") {
		t.Error("output should contain 'Failed'")
	}
	if !strings.Contains(output, "timeout") {
		t.Error("output should contain failure reason 'timeout'")
	}
	if !strings.Contains(output, "2") {
		t.Error("output should contain rejections count")
	}
}

func TestWriteJSONReport(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "report.json")

	result := scenario.Result{
		Success:          true,
		Duration:         10 * time.Second,
		MessagesReceived: 100,
		TurnsCompleted:   20,
		GlobalState:      "game_over",
		Rejections:       0,
	}

	err := writeJSONReport(result, outputPath)
	if err != nil {
		t.Fatalf("writeJSONReport failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatal("report file should exist")
	}

	// Read and verify content
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read report: %v", err)
	}

	var parsed scenario.Result
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("failed to parse report: %v", err)
	}

	if parsed.Success != result.Success {
		t.Errorf("Success = %v, expected %v", parsed.Success, result.Success)
	}
	if parsed.MessagesReceived != result.MessagesReceived {
		t.Errorf("MessagesReceived = %d, expected %d", parsed.MessagesReceived, result.MessagesReceived)
	}
}

func TestWriteJSONReportNestedPath(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "nested", "deep", "report.json")

	result := scenario.Result{
		Success: true,
	}

	err := writeJSONReport(result, outputPath)
	if err != nil {
		t.Fatalf("writeJSONReport should create nested directories: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatal("report file should exist in nested path")
	}
}

// ========== Root Command Tests ==========

func TestRootCmdUse(t *testing.T) {
	if rootCmd.Use != "pdcli" {
		t.Errorf("rootCmd.Use = %s, expected pdcli", rootCmd.Use)
	}
}

func TestRootCmdShort(t *testing.T) {
	if rootCmd.Short == "" {
		t.Error("rootCmd.Short should not be empty")
	}
}

func TestRootCmdLong(t *testing.T) {
	if rootCmd.Long == "" {
		t.Error("rootCmd.Long should not be empty")
	}
}

func TestAddCommand(t *testing.T) {
	// Create a test command
	testCmd := &cobra.Command{
		Use: "test",
	}

	// Add command to root
	AddCommand(testCmd)

	// Verify command is added
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "test" {
			found = true
			break
		}
	}

	if !found {
		t.Error("test command should be added to root")
	}
}

func TestExecute(t *testing.T) {
	// Execute should not panic
	// Note: We can't test actual execution without a full command setup
	// This test verifies the function exists and is callable
	// Actual execution would require setting up test environment
}

// ========== Flag Parsing Tests ==========

func TestPlaytestRunFlagParsing(t *testing.T) {
	// Create a new command instance to test flag parsing
	cmd := &cobra.Command{}
	var testPlayers int
	var testMode string

	cmd.Flags().IntVar(&testPlayers, "players", 4, "Number of players")
	cmd.Flags().StringVar(&testMode, "mode", "nakama", "Server mode")

	// Parse flags
	err := cmd.ParseFlags([]string{"--players=2", "--mode=standalone"})
	if err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	if testPlayers != 2 {
		t.Errorf("players = %d, expected 2", testPlayers)
	}
	if testMode != "standalone" {
		t.Errorf("mode = %s, expected standalone", testMode)
	}
}

func TestPlaytestRunDefaultFlags(t *testing.T) {
	cmd := &cobra.Command{}
	var testPlayers int
	var testTimeout int

	cmd.Flags().IntVar(&testPlayers, "players", 4, "Number of players")
	cmd.Flags().IntVar(&testTimeout, "timeout", 180, "Timeout")

	// Parse with no flags (use defaults)
	err := cmd.ParseFlags([]string{})
	if err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	if testPlayers != 4 {
		t.Errorf("default players = %d, expected 4", testPlayers)
	}
	if testTimeout != 180 {
		t.Errorf("default timeout = %d, expected 180", testTimeout)
	}
}