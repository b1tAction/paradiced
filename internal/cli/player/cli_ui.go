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
	"github.com/b1tAction/paradiced/pkg/gamelog"
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

	// Display entries if present
	if len(state.Entries) > 0 {
		fmt.Println("Entries:")
		for _, entry := range state.Entries {
			ui.displayLogEntry(entry)
		}
		fmt.Println()
	}

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

// OnMiniGameStart prompts user for mini-game data submission.
func (ui *CLIUIAdapter) OnMiniGameStart(ctx context.Context, start *model.MiniGameStart) map[string]interface{} {
	fmt.Println()
	fmt.Println("========== Mini Game Start ==========")
	fmt.Printf("Game Type: %s\n", start.GameType)
	fmt.Printf("Participants: %d\n", len(start.Players))
	fmt.Println()

	// Find my index for default values
	myIndex := 0
	for i, uid := range start.Players {
		if uid == ui.userID {
			myIndex = i
			break
		}
	}

	// Generate game_data based on game_type
	switch start.GameType {
	case "count_seconds":
		// count_seconds: prompt for estimated elapsed time (target: 5.0 seconds)
		defaultElapsed := 5.0 + float64(myIndex) * 0.3
		elapsed := 0.0
		for {
			fmt.Printf("Enter your estimated seconds (target: 5.0, default %.1f): ", defaultElapsed)
			if !ui.reader.Scan() {
				elapsed = defaultElapsed
				break
			}
			input := strings.TrimSpace(ui.reader.Text())
			if input == "" {
				elapsed = defaultElapsed
				break
			}
			val, err := strconv.ParseFloat(input, 64)
			if err != nil {
				fmt.Println("Invalid time, enter a number")
				continue
			}
			elapsed = val
			break
		}
		return map[string]interface{}{
			"elapsed": elapsed,
		}

	case "dice_race":
		// dice_race: prompt for two dice values (1-6)
		defaultD1 := 6 - myIndex
		if defaultD1 < 1 {
			defaultD1 = 1
		}
		defaultD2 := 5 - myIndex
		if defaultD2 < 1 {
			defaultD2 = 1
		}

		d1 := defaultD1
		for {
			fmt.Printf("Enter dice1 (1-6, default %d): ", defaultD1)
			if !ui.reader.Scan() {
				d1 = defaultD1
				break
			}
			input := strings.TrimSpace(ui.reader.Text())
			if input == "" {
				d1 = defaultD1
				break
			}
			val, err := strconv.Atoi(input)
			if err != nil || val < 1 || val > 6 {
				fmt.Println("Invalid dice value, enter a number 1-6")
				continue
			}
			d1 = val
			break
		}

		d2 := defaultD2
		for {
			fmt.Printf("Enter dice2 (1-6, default %d): ", defaultD2)
			if !ui.reader.Scan() {
				d2 = defaultD2
				break
			}
			input := strings.TrimSpace(ui.reader.Text())
			if input == "" {
				d2 = defaultD2
				break
			}
			val, err := strconv.Atoi(input)
			if err != nil || val < 1 || val > 6 {
				fmt.Println("Invalid dice value, enter a number 1-6")
				continue
			}
			d2 = val
			break
		}

		return map[string]interface{}{
			"dice1": d1,
			"dice2": d2,
			"score": d1 + d2,
		}

	default:
		// coin_flip and others: prompt for score
		score := 0
		for {
			fmt.Printf("Enter your score (default %d): ", (len(start.Players) - myIndex) * 100)
			if !ui.reader.Scan() {
				score = (len(start.Players) - myIndex) * 100
				break
			}
			input := strings.TrimSpace(ui.reader.Text())
			if input == "" {
				score = (len(start.Players) - myIndex) * 100
				break
			}
			val, err := strconv.Atoi(input)
			if err != nil {
				fmt.Println("Invalid score, enter a number")
				continue
			}
			score = val
			break
		}

		return map[string]interface{}{
			"score": score,
		}
	}
}

// OnMiniGameResult displays mini-game result.
func (ui *CLIUIAdapter) OnMiniGameResult(ctx context.Context, result *model.MiniGameResult) {
	fmt.Println()
	fmt.Println("========== Mini Game Final Ranking ==========")
	for _, entry := range result.Rankings {
		myRank := ""
		// Server may send either internal PlayerID or Nakama UserID.
		if entry.PlayerID == ui.playerID || entry.PlayerID == ui.userID {
			myRank = " (You)"
		}
		// Use DisplayName for display, fallback to PlayerID if empty
		displayName := entry.DisplayName
		if displayName == "" {
			displayName = entry.PlayerID
		}
		fmt.Printf("Rank %d: %s%s\n", entry.Rank, displayName, myRank)
	}
	fmt.Println("======================================")
	fmt.Println()
}

// OnGameOver displays game over information.
func (ui *CLIUIAdapter) OnGameOver(ctx context.Context, gameOver *model.GameOver) {
	fmt.Println()
	fmt.Println("========== GAME OVER ==========")
	winnerName := ui.resolveDisplayName(gameOver.WinnerID)
	fmt.Printf("Winner: %s (defeated the Boss!)\n", winnerName)
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

func (ui *CLIUIAdapter) displayLogEntry(entry gamelog.LogEntry) {
	// Resolve display name for target (prefer DisplayName from cached StateSync)
	targetName := ui.resolveDisplayName(entry.Target)
	source := entry.Source

	// Non-action entries (state, mini_game, boss, decision) use Type field for classification.
	// Action entries use ActionType field. State entries have empty ActionType but metadata has from/to.
	switch {
	case entry.Type == constants.EntryTypeState:
		from := entry.Metadata.GetStringOrDefault("from", "?")
		to := entry.Metadata.GetStringOrDefault("to", "?")
		fmt.Printf("  [state] %s: %s -> %s\n", targetName, from, to)

	case entry.ActionType == "damage":
		hpChange := entry.Metadata.GetIntOrDefault("hp_change", 0)
		blockedBy := entry.Metadata.GetStringOrDefault("blocked_by", "")
		piercing := entry.Metadata.GetBoolOrDefault("piercing", false)
		extra := ""
		if blockedBy != "" {
			extra += fmt.Sprintf(" [blocked by %s]", blockedBy)
		}
		if piercing {
			extra += " [piercing]"
		}
		fmt.Printf("  [damage] %s HP%d from %s%s\n", targetName, hpChange, source, extra)

	case entry.ActionType == "heal":
		hpChange := entry.Metadata.GetIntOrDefault("hp_change", 0)
		fmt.Printf("  [heal] %s HP+%d from %s\n", targetName, hpChange, source)

	case entry.ActionType == "modify_lp":
		lpChange := entry.Metadata.GetIntOrDefault("lp_change", 0)
		sign := "+"
		if lpChange < 0 {
			sign = ""
		}
		fmt.Printf("  [modify_lp] %s LP%s%d from %s\n", targetName, sign, lpChange, source)

	case entry.ActionType == "move":
		steps := entry.Metadata.GetIntOrDefault("steps", 0)
		startPos := entry.Metadata.GetIntOrDefault("start_pos", 0)
		endPos := entry.Metadata.GetIntOrDefault("end_pos", 0)
		fmt.Printf("  [move] %s %d steps (pos %d -> %d) from %s\n", targetName, steps, startPos, endPos, source)

	case entry.ActionType == "add_buff":
		buffType := entry.Metadata.GetStringOrDefault("buff_type", "")
		duration := entry.Metadata.GetIntOrDefault("duration", 0)
		fmt.Printf("  [add_buff] %s gained %s (duration: %d) from %s\n", targetName, buffType, duration, source)

	case entry.ActionType == "remove_buff":
		buffType := entry.Metadata.GetStringOrDefault("buff_type", "")
		fmt.Printf("  [remove_buff] %s lost %s from %s\n", targetName, buffType, source)

	case entry.ActionType == "draw_event":
		eventType := entry.Metadata.GetStringOrDefault("event_type", "")
		fmt.Printf("  [draw_event] %s drew event %s\n", targetName, eventType)

	case entry.ActionType == "draw_item":
		itemType := entry.Metadata.GetStringOrDefault("item_type", "")
		fmt.Printf("  [draw_item] %s drew item %s\n", targetName, itemType)

	case entry.ActionType == "teleport":
		fromPos := entry.Metadata.GetIntOrDefault("from_pos", 0)
		toPos := entry.Metadata.GetIntOrDefault("to_pos", 0)
		fmt.Printf("  [teleport] %s pos %d -> %d from %s\n", targetName, fromPos, toPos, source)

	case entry.ActionType == "steal_buff":
		stolenBy := entry.Metadata.GetStringOrDefault("stolen_by", "")
		buffType := entry.Metadata.GetStringOrDefault("buff_type", "")
		stolenByName := ui.resolveDisplayName(stolenBy)
		fmt.Printf("  [steal_buff] %s stolen %s by %s\n", targetName, buffType, stolenByName)

	case entry.ActionType == "fell_down":
		position := entry.Metadata.GetIntOrDefault("position", 0)
		hpChange := entry.Metadata.GetIntOrDefault("hp_change", 0)
		fmt.Printf("  [fell_down] %s fell at pos %d HP%d from %s\n", targetName, position, hpChange, source)

	case entry.ActionType == "respawn":
		checkpointPos := entry.Metadata.GetIntOrDefault("checkpoint_pos", 0)
		fmt.Printf("  [respawn] %s respawn at pos %d from %s\n", targetName, checkpointPos, source)

	case entry.ActionType == "dice_roll":
		diceType := entry.Metadata.GetStringOrDefault("dice_type", "")
		diceSteps := entry.Metadata.GetIntOrDefault("dice_steps", 0)
		fmt.Printf("  [dice_roll] %s rolled %s dice: %d steps\n", targetName, diceType, diceSteps)

	case entry.ActionType == "boss_damage":
		damage := entry.Metadata.GetIntOrDefault("damage", 0)
		isCrit := entry.Metadata.GetBoolOrDefault("is_crit", false)
		bossHP := entry.Metadata.GetIntOrDefault("boss_remaining_hp", 0)
		critMark := ""
		if isCrit {
			critMark = " [CRIT!]"
		}
		fmt.Printf("  [boss_damage] Boss HP-%d%s (remaining: %d) by %s\n",
			damage, critMark, bossHP, source)

	case entry.ActionType == "boss_attack":
		attackType := entry.Metadata.GetStringOrDefault("attack_type", "")
		damage := entry.Metadata.GetIntOrDefault("damage", 0)
		fmt.Printf("  [boss_attack] %s: Boss attacked %s (%s) HP-%d\n",
			source, targetName, attackType, damage)

	case entry.ActionType == "boss_skill":
		skillType := entry.Metadata.GetStringOrDefault("skill_type", "")
		targetsStr := entry.Metadata.GetStringOrDefault("targets", "")
		fmt.Printf("  [boss_skill] Boss used %s on %s\n", skillType, targetsStr)

	default:
		// Generic fallback for unknown entry types
		typeStr := string(entry.Type)
		if entry.ActionType != "" {
			typeStr = entry.ActionType
		}
		fmt.Printf("  [%s] target=%s source=%s\n", typeStr, targetName, source)
	}
}

// resolveDisplayName maps a player ID to a display name using cached StateSync.
func (ui *CLIUIAdapter) resolveDisplayName(playerID string) string {
	// Boss player has a fixed display name
	if playerID == constants.BossPlayerUUID {
		return "Boss"
	}
	if ui.stateSync == nil {
		return playerID
	}
	for _, player := range ui.stateSync.Players {
		if player.PlayerID == playerID {
			name := player.DisplayName
			if name == "" {
				name = player.PlayerID
			}
			return name
		}
	}
	return playerID
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
			fmt.Println("Host commands: 1=start, 2=wait")
			fmt.Println("==================================")
			fmt.Println()

			for {
				fmt.Print("Enter command (1=start, 2=wait): ")

				if !ui.reader.Scan() {
					// EOF/interrupt -> keep waiting safely
					return false
				}

				input := strings.TrimSpace(strings.ToLower(ui.reader.Text()))
				switch input {
				case "1", "start", "s":
					return true
				case "2", "wait", "w":
					fmt.Println("[Waiting] Continue waiting for players...")
					return false
				default:
					fmt.Println("Invalid choice, please enter 1 (start) or 2 (wait)")
				}
			}
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

// OnStartGameAck displays game start acknowledgment with map info.
func (ui *CLIUIAdapter) OnStartGameAck(ctx context.Context, ack *model.StartGameAck) {
	fmt.Println()
	fmt.Println("========== Game Started ==========")
	fmt.Printf("Map: %d cells, start=%d, end=%d\n",
		ack.MapConfig.Length, ack.MapConfig.StartIndex, ack.MapConfig.EndIndex)

	// Show cell type summary
	cellTypes := make(map[string]int)
	for _, cell := range ack.MapConfig.Cells {
		cellTypes[cell.CellType]++
	}
	fmt.Println("Cell types:")
	for ct, count := range cellTypes {
		fmt.Printf("  %s: %d\n", ct, count)
	}
	fmt.Println("==================================")
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
// PlayerID now equals the frontend userID, so we can directly match.
func (ui *CLIUIAdapter) findMyPlayerID(state *model.StateSync) string {
	for _, player := range state.Players {
		if player.PlayerID == ui.userID {
			return player.PlayerID
		}
	}
	return ""
}

// displayPlayers displays all players in the game.
func (ui *CLIUIAdapter) displayPlayers(state *model.StateSync) {
	fmt.Println("Players:")
	for _, player := range state.Players {
		// Boss player has special display format
		if player.IsBoss {
			bossStatus := ""
			if player.IsDead {
				bossStatus = " [DEFEATED]"
			}
			fmt.Printf("  * Boss HP:%d/50 Pos:%d%s\n",
				player.HP, player.Position, bossStatus)
			continue
		}

		isMe := player.PlayerID == ui.userID
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
