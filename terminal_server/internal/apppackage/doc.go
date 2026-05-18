// Package apppackage builds and validates canonical .tap application archives.
//
// A .tap file is a zstd-compressed tar archive with a required manifest.toml
// and optional ed25519 signature bundle. BuildTapFromDir assembles an archive
// from a source directory; VerifyTap validates structure, schema, and signature.
// The VerifiedTap type is the trusted representation passed downstream to appruntime.
package apppackage
