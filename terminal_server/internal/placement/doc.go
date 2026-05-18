// Package placement resolves semantic target scopes (zone/role/nearest)
// to concrete device IDs.
//
// Scenarios express targets as placement queries rather than hard-coded device
// IDs; the Engine translates those queries by consulting device.Manager and the
// active claim snapshot. The three primary resolution modes are DevicesInZone,
// DevicesWithRole, and NearestWith (proximity-based capability lookup).
package placement
