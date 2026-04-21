// Package player provides interactive player functionality for CLI.
package player

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/b1tAction/paradiced/internal/cli/model"
	"github.com/b1tAction/paradiced/pkg/constants"
)

// CLIUIAdapter implements PlayerUIAdapter using simple text output.
// Uses fmt.Printf for output and bufio.Scanner for input.
type CLIUIAdapter struct {
	reader    *bufio.Scanner
	stateSync *model.StateSync
	userID    string
	playerID  string
	verbose   bool
	faction   constants.Faction
}

// NewCLIUIAdapter creates a new CLI UI adapter.
func NewCLIUIAdapter(userID string, faction constants.Faction, verbose bool) *CLIUIAdapter {
	return &CLIUIAdapter{
		reader:  bufio.NewScanner(os.Stdin),
		userID:  userID,
		faction: faction,
		verbose: verbose,
	}
}

// Clear clears the UI state.
func (ui *CLIUIAdapter) Clear() {
	ui.stateSync = nil
}

// GetCurrentState returns the current game state.
func (ui *CLIUIAdapter) GetCurrentState() *model.StateSync {
	return ui.stateSync
}

// GetMyPlayerID returns the current player's game ID.
func (ui *CLIUIAdapter) GetMyPlayerID() string {
	return ui.playerID
}

// GetUserID returns the Nakama user ID.
func (ui *CLIUIAdapter) GetUserID() string {
	return ui.userID
}

// OnStateSync displays game state update.
func (ui *CLIUIAdapter) OnStateSync(ctx context.Context, state *model.StateSync) {
	prev := ui.stateSync
	ui.stateSync = state
	ui.playerID = ui.findMyPlayerID(state)

	if !ui.verbose {
		stateChanged := prev == nil ||
			prev.GlobalState != state.GlobalState ||
			prev.TurnState != state.TurnState ||
			prev.Round != state.Round ||
			prev.Turn != state.Turn ||
			prev.CurrentPlayerID != state.CurrentPlayerID

		if stateChanged {
			myTurnMark := ""
			if state.CurrentPlayerID != "" && state.CurrentPlayerID == ui.playerID {
				myTurnMark = " [Your Turn]"
			}
			fmt.Printf("\n[State] %s/%s R%d T%d%s\n",
				state.GlobalState, state.TurnState, state.Round, state.Turn, myTurnMark)
		}
		return
	}

	// Display state header
	fmt.Println()
	fmt.Printf("========== State Update ==========\n")
	fmt.Printf("Global State: %s\n", state.GlobalState)
	fmt.Printf("Turn State: %s\n", state.TurnState)
	fmt.Printf("Round: %d | Turn: %d\n", state.Round, state.Turn)
	fmt.Printf("Current Player: %s\n", state.CurrentPlayerID)
	fmt.Println()

	// Display all players
	ui.displayPlayers(state)

	fmt.Println("==================================")
	fmt.Println()
}

// OnAvailable prompts user to choose an action.
func (ui *CLIUIAdapter) OnAvailable(ctx context.Context, available *model.Available) PlayerAction {
	// Check if it's my turn
	isMyTurn := ui.stateSync != nil && ui.stateSync.CurrentPlayerID == ui.playerID

	fmt.Println()
	if isMyTurn {
		fmt.Println(">>> Your Turn! <<<")
	} else {
		fmt.Println(">>> Available Actions Received (may not be your turn) <<<")
	}
	fmt.Println()

	// Display available options
	ui.displayAvailableOptions(available)

	// Prompt for input
	for {
		fmt.Printf("Select action (enter number): ")

		if !ui.reader.Scan() {
			return PlayerAction{Type: ActionNone}
		}

		input := strings.TrimSpace(ui.reader.Text())
		choice, err := strconv.Atoi(input)
		if err != nil {
			fmt.Println("Invalid input, please enter a number")
			continue
		}

		action := ui.parseChoice(choice, available)
		if action.IsValid() {
			return action
		}

		fmt.Println("Invalid choice, please try again")
	}
}

// OnMiniGameStart prompts user for mini-game participation.
func (ui *CLIUIAdapter) OnMiniGameStart(ctx context.Context, start *model.MiniGameStart) int {
	fmt.Println()
	fmt.Println("========== Mini Game Start ==========")
	fmt.Printf("Game Type: %s\n", start.GameType)
	fmt.Printf("Participants: %d\n", len(start.Players))
	fmt.Println()

	// Find my index
	myIndex := 0
	for i, uid := range start.Players {
		if uid == ui.userID {
			myIndex = i
			break
		}
	}

	// Prompt for rank
	for {
		fmt.Printf("Enter your rank (1-%d): ", len(start.Players))

		if !ui.reader.Scan() {
			// Default to index-based rank
			return myIndex + 1
		}

		input := strings.TrimSpace(ui.reader.Text())
		rank, err := strconv.Atoi(input)
		if err != nil || rank < 1 || rank > len(start.Players) {
			fmt.Printf("Invalid rank, enter 1-%d\n", len(start.Players))
			continue
		}

		return rank
	}
}

// OnMiniGameResult displays mini-game result.
func (ui *CLIUIAdapter) OnMiniGameResult(ctx context.Context, result *model.MiniGameResult) {
	fmt.Println()
	fmt.Println("========== Mini Game Result ==========")
	for _, entry := range result.Rankings {
		myRank := ""
		if entry.PlayerID == ui.playerID {
			myRank = " (You)"
		}
		fmt.Printf("Rank %d: %s%s\n", entry.Rank, entry.PlayerID, myRank)
	}
	fmt.Println("======================================")
	fmt.Println()
}

// OnGameOver displays game over information.
func (ui *CLIUIAdapter) OnGameOver(ctx context.Context, gameOver *model.GameOver) {
	fmt.Println()
	fmt.Println("========== GAME OVER ==========")
	fmt.Printf("Winner: %s\n", gameOver.WinnerID)
	fmt.Println()

	fmt.Println("Player Stats:")
	for _, stat := range gameOver.Stats {
		myStats := ""
		if ui.playerID != "" && stat.PlayerID == ui.playerID {
			myStats = " (You)"
		}
		fmt.Printf("  %s%s: RoundsWon=%d, Events=%d, Items=%d\n",
			stat.PlayerID, myStats, stat.RoundsWon, stat.EventsDrawn, stat.ItemsUsed)
	}

	fmt.Println("================================")
	fmt.Println()
}

// OnActionRejected displays action rejection notification.
func (ui *CLIUIAdapter) OnActionRejected(ctx context.Context, rejected *model.ActionRejected) {
	fmt.Println()
	fmt.Println("!!! Action Rejected !!!")
	fmt.Printf("OpCode: %d\n", rejected.OpCode)
	fmt.Printf("ErrorCode: %d\n", rejected.ErrorCode)
	fmt.Printf("Reason: %s\n", rejected.Reason)
	fmt.Printf("Message: %s\n", rejected.Message)
	fmt.Println()
}

// OnTurnSync displays turn update.
func (ui *CLIUIAdapter) OnTurnSync(ctx context.Context, turnSync *model.TurnSync) {
	if ui.verbose {
		fmt.Println()
		fmt.Println("---------- Turn Sync ----------")
		fmt.Printf("Round: %d | Turn: %d\n", turnSync.Round, turnSync.Turn)
		fmt.Printf("Current Player: %s\n", turnSync.CurrentPlayerID)
		if len(turnSync.Entries) > 0 {
			fmt.Printf("Log Entries: %d\n", len(turnSync.Entries))
		}
		fmt.Println("------------------------------")
		fmt.Println()
	}
}

// OnFullSync displays full sync (reconnection).
func (ui *CLIUIAdapter) OnFullSync(ctx context.Context, state *model.StateSync) {
	fmt.Println()
	fmt.Println("========== Full Sync (Reconnection) ==========")
	ui.OnStateSync(ctx, state)
}

// OnError displays error message.
func (ui *CLIUIAdapter) OnError(err error) {
	fmt.Println()
	fmt.Printf("!!! Error: %v !!!\n", err)
	fmt.Println()
}

// OnWaiting displays waiting status.
func (ui *CLIUIAdapter) OnWaiting(message string, playerCount int, maxPlayers int) {
	fmt.Println()
	fmt.Printf("[Waiting] %s\n", message)
	fmt.Printf("Players: %d/%d\n", playerCount, maxPlayers)
	fmt.Println()
}

// OnWaitingSync displays waiting room status for the host.
func (ui *CLIUIAdapter) OnWaitingSync(ctx context.Context, waiting *model.WaitingSync) bool {
	fmt.Println()
	fmt.Println("========== Waiting Room ==========")
	// Room ID is shown earlier in OnMatchCreated, skip if empty in WaitingSync
	if waiting.MatchID != "" {
		fmt.Printf("Room ID: %s\n", waiting.MatchID)
	}
	fmt.Printf("Host: %s\n", waiting.HostUserID)
	fmt.Printf("Players: %d/%d (min: %d)\n", waiting.PlayerCount, waiting.MaxPlayers, waiting.MinPlayers)
	fmt.Println()

	fmt.Println("Players in room:")
	for _, player := range waiting.Players {
		hostMark := ""
		meMark := ""
		if player.IsHost {
			hostMark = " [Host]"
		}
		if player.UserID == ui.userID {
			meMark = " (You)"
		}
		// Use DisplayName for display, fallback to UserID if empty
		displayName := player.DisplayName
		if displayName == "" {
			displayName = player.UserID
		}
		fmt.Printf("  %s (%s)%s%s\n", displayName, player.Faction, hostMark, meMark)
	}
	fmt.Println()

	if waiting.CanStart {
		// Only host can start the game
		if ui.userID == waiting.HostUserID {
			fmt.Printf(">>> %s <<<\n", waiting.Message)
			fmt.Println()
			fmt.Print("Type 'start' to begin the game, or 'wait' for more players: ")

			if !ui.reader.Scan() {
				return false
			}

			input := strings.TrimSpace(ui.reader.Text())
			if input == "start" {
				fmt.Println()
				fmt.Println("Starting game...")
				return true
			}
			fmt.Println()
			fmt.Println("Waiting for more players...")
			return false
		} else {
			// Non-host players see waiting message
			fmt.Printf("[Waiting] Host (%s) can start the game\n", waiting.HostUserID)
			fmt.Println("==================================")
			fmt.Println()
			return false
		}
	} else {
		fmt.Printf("[Waiting] %s\n", waiting.Message)
		fmt.Println("==================================")
		fmt.Println()
		return false
	}
}

// OnConnected displays connection success.
func (ui *CLIUIAdapter) OnConnected(matchID string) {
	fmt.Println()
	fmt.Printf(">>> Connected to room: %s <<<\n", matchID)
	fmt.Println()
}

// OnPlayerJoined displays player join notification.
func (ui *CLIUIAdapter) OnPlayerJoined(userID string, faction string) {
	fmt.Println()
	fmt.Printf("[Player Joined] %s (%s)\n", userID, faction)
}

// OnMatchCreated displays match creation success.
func (ui *CLIUIAdapter) OnMatchCreated(matchID string) {
	fmt.Println()
	fmt.Println("========== Room Created ==========")
	fmt.Printf("Room ID: \"%s\"\n", matchID)
	fmt.Println("(Include the trailing dot when joining)")
	fmt.Println("Other players can join using this ID")
	fmt.Println("==================================")
	fmt.Println()
}

// ========== Helper Methods ==========

// findMyPlayerID finds the player's game ID from StateSync.
func (ui *CLIUIAdapter) findMyPlayerID(state *model.StateSync) string {
	for _, player := range state.Players {
		if player.ClientID == ui.userID {
			return player.PlayerID
		}
	}
	return ""
}

// displayPlayers displays all players in the game.
func (ui *CLIUIAdapter) displayPlayers(state *model.StateSync) {
	fmt.Println("Players:")
	for _, player := range state.Players {
		isMe := player.ClientID == ui.userID
		isCurrent := player.PlayerID == state.CurrentPlayerID

		prefix := "  "
		if isMe {
			prefix = "  * "
		}
		if isCurrent {
			prefix = "  > "
		}
		if isMe && isCurrent {
			prefix = "  *> "
		}

		// Use DisplayName for display, fallback to PlayerID if empty
		displayName := player.DisplayName
		if displayName == "" {
			displayName = player.PlayerID
		}

		fmt.Printf("%s%s (%s) Pos:%d HP:%d LP:%d\n",
			prefix, displayName, player.Faction, player.Position, player.HP, player.LP)

		// Show buffs
		if len(player.Buffs) > 0 {
			buffNames := make([]string, 0)
			for _, buff := range player.Buffs {
				buffNames = append(buffNames, fmt.Sprintf("%s(%d)", buff.Name, buff.Duration))
			}
			fmt.Printf("	Buffs: %s\n", strings.Join(buffNames, ", "))
		}

		// Show items
		if len(player.Items) > 0 {
			itemNames := make([]string, 0)
			for _, item := range player.Items {
				itemNames = append(itemNames, item.Name)
			}
			fmt.Printf("	Items: %s\n", strings.Join(itemNames, ", "))
		}

		// Show my detailed status
		if isMe && ui.verbose {
			fmt.Printf("	Charge: %d\n", player.Charge)
			fmt.Printf("	Status: Dead=%v, SkipTurn=%v\n", player.IsDead, player.SkipTurn)
		}
	}
}

// displayAvailableOptions displays available action options.
func (ui *CLIUIAdapter) displayAvailableOptions(available *model.Available) {
	fmt.Println("Available actions:")
	fmt.Println("  1. Roll dice to move")

	// Display items
	for i, item := range available.Items {
		fmt.Printf("  %d. Use item: %s (%s)\n", i+2, item.Name, item.Type)
	}

	// Display view status option
	fmt.Printf("  %d. View detailed status\n", len(available.Items)+2)

	fmt.Println()

	// Show additional info
	if available.CanUseSkill {
		fmt.Println("  [Faction skill available]")
	}
	if available.DiceType != "normal" {
		fmt.Printf("  [Dice type: %s]\n", available.DiceType)
	}
}

// parseChoice parses user choice to PlayerAction.
func (ui *CLIUIAdapter) parseChoice(choice int, available *model.Available) PlayerAction {
	// Choice 1 is always roll dice
	if choice == 1 {
		return NewRollDiceAction()
	}

	// Choice 2 to len(items)+1 are items
	itemCount := len(available.Items)
	if choice >= 2 && choice <= itemCount+1 {
		item := available.Items[choice-2]
		return NewUseItemAction(item.ID)
	}

	// Choice len(items)+2 is view status
	if choice == itemCount+2 {
		return NewViewStatusAction()
	}

	return PlayerAction{Type: ActionNone}
}
