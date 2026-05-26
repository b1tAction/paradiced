// Package net provides protocol types for MiniGameRequest RPC.
package net

// MiniGameRequestPayload is the unified request payload for the minigame_request RPC.
// It supports two modes:
//   - "judge": offline rank calculation from submitted game_data
//   - "online": trigger an online mini-game session via MatchSignal
type MiniGameRequestPayload struct {
	Mode        string                    `json:"mode"`        // "judge" or "online"
	GameType    string                    `json:"game_type"`   // e.g. "dice_race", "dilemma_race"
	Submissions []MiniGameJudgeSubmission `json:"submissions"` // for judge mode: player game_data
	MatchID     string                    `json:"match_id"`    // for online mode: target Nakama match
}

// MiniGameJudgeSubmission represents a single player's game data for offline judge mode.
type MiniGameJudgeSubmission struct {
	PlayerID    string                 `json:"player_id"`
	DisplayName string                 `json:"display_name"`
	GameData    map[string]interface{} `json:"game_data"`
}

// MiniGameJudgeResponse is the response for the offline judge mode.
// Contains calculated rankings and dice type assignments.
type MiniGameJudgeResponse struct {
	Rankings        []MiniGameJudgeRankingEntry `json:"rankings"`
	DiceAssignments map[string]string           `json:"dice_assignments"` // player_id -> dice_type
}

// MiniGameJudgeRankingEntry represents a single player's ranking in judge mode.
type MiniGameJudgeRankingEntry struct {
	PlayerID    string                 `json:"player_id"`
	DisplayName string                 `json:"display_name"`
	Rank        int                    `json:"rank"`
	GameData    map[string]interface{} `json:"game_data,omitempty"`
}

// MiniGameOnlineResponse is the response for the online trigger mode.
type MiniGameOnlineResponse struct {
	Success    bool        `json:"success"`
	MatchID    string      `json:"match_id"`
	Connection *MiniGameConn `json:"connection,omitempty"`
}