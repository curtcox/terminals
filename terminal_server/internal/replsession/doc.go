// Package replsession manages REPL session lifecycle and attachment state.
//
// A REPL session ties a terminal device to an active AI provider session.
// This package tracks open sessions, handles attach/detach transitions, and
// enforces that a session belongs to exactly one device at a time. The Service
// interface is the primary entry point; callers create, look up, and close
// sessions by session ID or device ID.
package replsession
