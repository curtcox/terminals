// Package appruntime loads and hot-reloads TAR/TAL application packages.
//
// It maintains a registry of named packages and their active versions, supports
// rollback to a prior version, and enforces capability permissions declared in
// the app manifest. Callers register packages via the Runtime interface; the
// engine validates kernel-API compatibility and runs any pending migration steps
// before making a new version active.
package appruntime
