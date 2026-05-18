// Package world stores calibration data and fused world-model entities.
//
// It owns the spatial pose and verification state for each device
// (manual, marker, audio-chirp, RF fingerprint, or mixed). Callers read and
// write WorldEntry records via the Model interface, which is backed by the
// io-router storage layer. The verification state drives how the UI surfaces
// location confidence to operators.
package world
