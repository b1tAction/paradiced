package hsm

import (
	"testing"
	"time"

	"github.com/b1tAction/paradiced/internal/core"
	"github.com/b1tAction/paradiced/internal/engine"
	engineaction "github.com/b1tAction/paradiced/internal/engine/action"
	"github.com/b1tAction/paradiced/internal/event"
	"github.com/b1tAction/paradiced/internal/gamemap"
	"github.com/b1tAction/paradiced/pkg/id"
)

func TestQueueDerived(t *testing.T) {
	game := engine.NewGame(id.NewGameID(), 0)
	mapEngine := gamemap.NewMapEngine(20)
	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 6, MaxLP: 3})
	player.HP = 3
	player.LP = 0
	game.AddPlayer(player)
	game.Log.StartTurn(1, 0, player.ID.UUID())

	actionCtx := engineaction.NewActionContext(game, game.Bus, mapEngine, game.Draw)
	triggerCtx := event.NewContext(player)
	triggerCtx.AddDerivedAction(engineaction.NewHealAction(player, 2, "Buff_Test"))
	triggerCtx.AddDerivedAction(engineaction.NewModifyLPAction(player, 1, "Buff_Test"))

	queueDerived(triggerCtx, actionCtx)

	if actionCtx.ActionQueue.Len() != 2 {
		t.Fatalf("ActionQueue len = %d, want 2", actionCtx.ActionQueue.Len())
	}
	if len(triggerCtx.GetDerivedActions()) != 0 {
		t.Fatalf("DerivedActions should be cleared after bridging, got %d", len(triggerCtx.GetDerivedActions()))
	}

	actionCtx.ProcessQueue()

	if player.HP != 5 {
		t.Fatalf("player.HP = %d, want 5", player.HP)
	}
	if player.LP != 1 {
		t.Fatalf("player.LP = %d, want 1", player.LP)
	}
}

func TestOnUserChoice_ProcessesDerivedActionsFromPendingDecisionContext(t *testing.T) {
	game := engine.NewGame(id.NewGameID(), 0)
	hsm := NewHSM(game)

	player := core.NewPlayer(core.PlayerConfig{ID: id.NewPlayerID(), MaxHP: 6})
	player.HP = 3
	game.AddPlayer(player)
	game.Log.StartTurn(1, 0, player.ID.UUID())

	hsm.SetTurnPlayer(player)

	actionCtx := engineaction.NewActionContext(game, game.Bus, nil, game.Draw)
	pendingCtx := event.NewContext(player)
	pendingCtx.Set("action_context", actionCtx)

	hsm.decision = event.NewDecision("confirm", []event.Option{
		{
			ID:    "ok",
			Label: "ok",
			Action: func(ctx *event.Context) error {
				ctx.AddDerivedAction(engineaction.NewHealAction(player, 2, "Decision_Test"))
				return nil
			},
		},
	})
	hsm.paused = true

	savedCtx := NewStateContext().WithHSM(hsm).WithPlayer(player)
	savedCtx.StartTime = time.Now()
	savedCtx.Set(KeyPendingCtx, pendingCtx)
	if err := hsm.stack.Push(&mockState{id: StateTurnUpkeep}, savedCtx); err != nil {
		t.Fatalf("push stack failed: %v", err)
	}

	err := hsm.OnUserChoice(0, NewStateContext().WithHSM(hsm).WithPlayer(player))
	if err != nil {
		t.Fatalf("OnUserChoice failed: %v", err)
	}

	if player.HP != 5 {
		t.Fatalf("player.HP = %d, want 5", player.HP)
	}
	if actionCtx.ActionQueue.Len() != 0 {
		t.Fatalf("ActionQueue should be empty after processing, got %d", actionCtx.ActionQueue.Len())
	}
	if savedCtx.HasKey(KeyPendingCtx) {
		t.Fatal("pending decision context should be cleared after user choice")
	}
}
