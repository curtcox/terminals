package io

func filterClaimsByActivation(claims []Claim, activationID string) []Claim {
	next := claims[:0]
	for _, existing := range claims {
		if existing.ActivationID == activationID {
			continue
		}
		next = append(next, existing)
	}
	return next
}

func filterClaimsByResource(claims []Claim, activationID, deviceID, resource string) []Claim {
	next := claims[:0]
	for _, existing := range claims {
		if existing.ActivationID == activationID && existing.DeviceID == deviceID && existing.Resource == resource {
			continue
		}
		next = append(next, existing)
	}
	return next
}

func setClaimSlice[K comparable](m map[K][]Claim, key K, claims []Claim) {
	if len(claims) == 0 {
		delete(m, key)
		return
	}
	m[key] = append([]Claim(nil), claims...)
}

func (m *ClaimManager) releaseOneClaimLocked(claim Claim) resourceKey {
	key := resourceKey{deviceID: claim.DeviceID, resource: claim.Resource}

	active := m.activeByResource[key]
	setClaimSlice(m.activeByResource, key, filterClaimsByActivation(active, claim.ActivationID))

	activeByAct := m.activeByAct[claim.ActivationID]
	setClaimSlice(m.activeByAct, claim.ActivationID, filterClaimsByResource(activeByAct, claim.ActivationID, claim.DeviceID, claim.Resource))

	parked := m.parkedByResource[key]
	setClaimSlice(m.parkedByResource, key, filterClaimsByActivation(parked, claim.ActivationID))

	parkedByAct := m.parkedByAct[claim.ActivationID]
	setClaimSlice(m.parkedByAct, claim.ActivationID, filterClaimsByResource(parkedByAct, claim.ActivationID, claim.DeviceID, claim.Resource))

	return key
}
