package hsm

import (
	engineaction "github.com/b1tAction/paradiced/internal/engine/action"
	"github.com/b1tAction/paradiced/internal/event"
)

const (
	// KeyPendingCtx stores the event.Context created at publish time
	// so decision callbacks can continue using the same context/action pipeline.
	KeyPendingCtx = "pending_ctx"
)

// queueDerived transfers derived actions from event context
// into ActionContext queue.
func queueDerived(triggerCtx *event.Context, actionCtx *engineaction.ActionContext) {
	if triggerCtx == nil || actionCtx == nil {
		return
	}

	for _, derived := range triggerCtx.GetDerivedActions() {
		execAction, ok := derived.(engineaction.Action)
		if !ok || execAction == nil {
			continue
		}
		actionCtx.PushDerivedAction(execAction)
	}

	triggerCtx.ClearDerivedActions()
}

// getActionCtx extracts ActionContext from event metadata.
func getActionCtx(triggerCtx *event.Context) *engineaction.ActionContext {
	if triggerCtx == nil {
		return nil
	}

	raw, ok := triggerCtx.Get("action_context")
	if !ok {
		return nil
	}

	actionCtx, _ := raw.(*engineaction.ActionContext)
	return actionCtx
}

// runDerived bridges and executes queued derived actions.
func runDerived(triggerCtx *event.Context) {
	actionCtx := getActionCtx(triggerCtx)
	if actionCtx == nil {
		return
	}

	queueDerived(triggerCtx, actionCtx)
	actionCtx.ProcessQueue()
}

// getPendingCtx returns the pending decision context from state context.
func getPendingCtx(ctx *StateContext) *event.Context {
	if ctx == nil {
		return nil
	}
	raw, ok := ctx.Get(KeyPendingCtx)
	if !ok {
		return nil
	}

	pendingCtx, _ := raw.(*event.Context)
	return pendingCtx
}
