// Package usecasevalidation provides a reusable test harness for running
// use-case validation scenarios against a real in-process server.
//
// The Harness type wires up a full server stack with fake AI and TTS providers,
// drives scenario steps via SimTerminal helpers, captures frame and audio
// evidence, and emits an EvidenceBundle to artifacts/usecase-validation/.
// Individual use-case tests (e.g., c1_test.go, aa2_test.go) import this
// package and call Harness methods to assert scenario outcomes.
package usecasevalidation
