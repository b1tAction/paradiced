// Package nakama provides Nakama match handler implementation for Paradiced.
package nakama

import (
	"context"
)

// NakamaMatchAdapter wraps Nakama's MatchAdapter to implement NakamaMatchWrapper.
// This implementation bridges between Paradiced's DispatcherAdapter and Nakama's
// MatchAdapter interface.
//
// When deployed to Nakama server, use this adapter with the real MatchAdapter:
//
//	func InitModule(ctx context.Context, logger runtime.Logger, initializer runtime.Initializer) error {
//	    return initializer.RegisterMatch("paradiced_match", func(ctx context.Context, logger runtime.Logger, match runtime.Match) (runtime.MatchHandler, error) {
//	        // Create wrapper
//	        adapter := NewNakamaMatchAdapter(match)
//	        // Create handler
//	        handler := NewNakamaMatchHandler(matchId, seed, 4, 100)
//	        handler.WithDispatcher(NewRealDispatcherAdapter(ctx, adapter))
//	        return handler, nil
//	    })
//	}
type NakamaMatchAdapter struct {
	match MatchAdapter
}

// MatchAdapter is the Nakama match interface stub.
// When deployed to Nakama server, this matches runtime.MatchAdapter.
type MatchAdapter interface {
	// Broadcast broadcasts data to presences.
	Broadcast(opCode int64, data []byte, presences []MatchPresence, reliability int) error

	// Send sends data to specific presences.
	Send(opCode int64, data []byte, presences []MatchPresence, reliability int) error

	// GetPresences returns current match presences.
	GetPresences() []MatchPresence
}

// MatchPresence is the Nakama presence stub.
// When deployed to Nakama server, this matches runtime.MatchPresence.
type MatchPresence interface {
	GetUserId() string
	GetSessionId() string
	GetNodeId() string
}

// NewNakamaMatchAdapter creates a new adapter wrapping a MatchAdapter.
func NewNakamaMatchAdapter(match MatchAdapter) *NakamaMatchAdapter {
	return &NakamaMatchAdapter{match: match}
}

// BroadcastData broadcasts message to all or specific presences.
func (a *NakamaMatchAdapter) BroadcastData(opCode int64, data []byte, presences []NakamaPresence, reliability int) error {
	// Convert NakamaPresence to MatchPresence
	matchPresences := make([]MatchPresence, len(presences))
	for i, p := range presences {
		matchPresences[i] = p.(MatchPresence)
	}
	return a.match.Broadcast(opCode, data, matchPresences, reliability)
}

// SendData sends message to specific presences.
func (a *NakamaMatchAdapter) SendData(opCode int64, data []byte, presences []NakamaPresence, reliability int) error {
	// Convert NakamaPresence to MatchPresence
	matchPresences := make([]MatchPresence, len(presences))
	for i, p := range presences {
		matchPresences[i] = p.(MatchPresence)
	}
	return a.match.Send(opCode, data, matchPresences, reliability)
}

// GetPresences returns current match presences.
func (a *NakamaMatchAdapter) GetPresences() []NakamaPresence {
	matchPresences := a.match.GetPresences()
	result := make([]NakamaPresence, len(matchPresences))
	for i, p := range matchPresences {
		result[i] = p
	}
	return result
}

// NakamaMatchHandlerWrapper wraps NakamaMatchHandler to implement Nakama's MatchHandler interface.
// This allows Paradiced's handler to be used directly as Nakama's match handler.
type NakamaMatchHandlerWrapper struct {
	handler *NakamaMatchHandler
	ctx     context.Context
	logger  Logger
	match   MatchAdapter
}

// Logger stub for Nakama logger interface.
type Logger interface {
	Info(msg string, fields ...interface{})
	Debug(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
}

// NewNakamaMatchHandlerWrapper creates a wrapper for Nakama MatchHandler interface.
func NewNakamaMatchHandlerWrapper(ctx context.Context, logger Logger, match MatchAdapter, matchID string, seed int64) *NakamaMatchHandlerWrapper {
	handler := NewNakamaMatchHandler(matchID, seed, 4, 100)
	adapter := NewNakamaMatchAdapter(match)
	dispatcher := NewRealDispatcherAdapter(ctx, adapter)
	handler.WithDispatcher(dispatcher)

	return &NakamaMatchHandlerWrapper{
		handler: handler,
		ctx:     ctx,
		logger:  logger,
		match:   match,
	}
}

// MatchInit implements Nakama MatchHandler.MatchInit.
func (w *NakamaMatchHandlerWrapper) MatchInit(ctx context.Context, logger Logger, match MatchAdapter) (interface{}, error) {
	// Update dispatcher with new match reference
	adapter := NewNakamaMatchAdapter(match)
	dispatcher := NewRealDispatcherAdapter(ctx, adapter)
	w.handler.WithDispatcher(dispatcher)
	w.ctx = ctx
	w.logger = logger
	w.match = match

	return w.handler, nil
}

// MatchJoin implements Nakama MatchHandler.MatchJoin.
func (w *NakamaMatchHandlerWrapper) MatchJoin(ctx context.Context, logger Logger, match MatchAdapter, presences []MatchPresence) error {
	for _, p := range presences {
		userID := p.GetUserId()
		// Update dispatcher presence map
		if dispatcher, ok := w.handler.dispatcher.(*RealDispatcherAdapter); ok {
			dispatcher.UpdatePresence(userID, p)
		}
		// Handle join logic
		w.handler.HandlePresenceJoin(userID, nil)
	}
	return nil
}

// MatchLeave implements Nakama MatchHandler.MatchLeave.
func (w *NakamaMatchHandlerWrapper) MatchLeave(ctx context.Context, logger Logger, match MatchAdapter, presences []MatchPresence) error {
	for _, p := range presences {
		userID := p.GetUserId()
		// Update dispatcher presence map
		if dispatcher, ok := w.handler.dispatcher.(*RealDispatcherAdapter); ok {
			dispatcher.RemovePresence(userID)
		}
		// Handle leave logic
		w.handler.HandlePresenceLeave(userID)
	}
	return nil
}

// MatchLoop implements Nakama MatchHandler.MatchLoop.
func (w *NakamaMatchHandlerWrapper) MatchLoop(ctx context.Context, logger Logger, match MatchAdapter) error {
	// Update dispatcher with fresh presences
	if dispatcher, ok := w.handler.dispatcher.(*RealDispatcherAdapter); ok {
		dispatcher.RefreshPresences()
	}
	return w.handler.MatchLoop(0) // delta time from Nakama tick
}

// MatchTerminate implements Nakama MatchHandler.MatchTerminate.
func (w *NakamaMatchHandlerWrapper) MatchTerminate(ctx context.Context, logger Logger, match MatchAdapter, graceSeconds int) error {
	w.handler.MatchStop()
	return nil
}

// HandleMessage implements Nakama MatchHandler.MatchMessage.
func (w *NakamaMatchHandlerWrapper) HandleMessage(ctx context.Context, logger Logger, match MatchAdapter, sender MatchPresence, data []byte) error {
	return w.handler.HandleMessage(sender.GetUserId(), data)
}