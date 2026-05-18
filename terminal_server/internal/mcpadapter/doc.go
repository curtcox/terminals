// Package mcpadapter bridges the MCP tool-call protocol to the REPL capability layer.
//
// It exposes the server's REPL capabilities as an MCP tool server, translating
// JSON-RPC tool-call requests into repl.Request values and routing responses
// back. The special repl_complete and repl_describe tools provide introspection;
// all other tool names are forwarded directly to the active REPL session identified
// by the session ID embedded in the request.
package mcpadapter
