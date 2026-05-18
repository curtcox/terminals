// Package replai manages sticky AI provider and model selection for REPL sessions.
//
// It stores the user's chosen provider and model per session ID and resolves
// them against the available ai.Provider implementations registered at startup.
// Callers use the Service interface to select, clear, and retrieve the active
// provider/model pair before routing a REPL request.
package replai
