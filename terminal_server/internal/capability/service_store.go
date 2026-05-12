package capability

import (
	"sort"
	"strings"
	"time"
)

// StorePut sets a key/value record in the given namespace.
// ttl <= 0 means the record does not expire.
func (s *Service) StorePut(namespace, key, value string, ttl time.Duration) StoreRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	ns := defaultIfBlank(namespace, "default")
	trimmedKey := strings.TrimSpace(key)
	storeKey := ns + ":" + trimmedKey
	existing, hasExisting := s.store[storeKey]
	record := StoreRecord{
		Namespace: ns,
		Key:       trimmedKey,
		Value:     value,
		UpdatedAt: s.now(),
	}
	if hasExisting {
		record.Binding = existing.Binding
	}
	if ttl > 0 {
		expiresAt := s.now().Add(ttl)
		record.ExpiresAt = &expiresAt
	}
	s.store[storeKey] = record
	s.appendRecentLocked("store", storeKey)
	return record
}

// StoreGet retrieves a record by namespace and key, returning false if not found.
func (s *Service) StoreGet(namespace, key string) (StoreRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	storeKey := defaultIfBlank(namespace, "default") + ":" + strings.TrimSpace(key)
	record, ok := s.store[storeKey]
	if !ok {
		return StoreRecord{}, false
	}
	if storeRecordExpired(record, s.now()) {
		delete(s.store, storeKey)
		return StoreRecord{}, false
	}
	return record, ok
}

// StoreList returns all records in the given namespace sorted by key.
func (s *Service) StoreList(namespace string) []StoreRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	ns := defaultIfBlank(namespace, "default")
	out := make([]StoreRecord, 0)
	now := s.now()
	for key, record := range s.store {
		if storeRecordExpired(record, now) {
			delete(s.store, key)
			continue
		}
		if record.Namespace == ns {
			out = append(out, record)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// StoreDelete removes a record by namespace and key.
func (s *Service) StoreDelete(namespace, key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	storeKey := defaultIfBlank(namespace, "default") + ":" + strings.TrimSpace(key)
	if _, ok := s.store[storeKey]; !ok {
		return false
	}
	delete(s.store, storeKey)
	s.appendRecentLocked("store", storeKey+" deleted")
	return true
}

// StoreNamespaces returns namespace inventory sorted by namespace.
func (s *Service) StoreNamespaces() []StoreNamespaceSummary {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	counts := map[string]int{}
	for key, record := range s.store {
		if storeRecordExpired(record, now) {
			delete(s.store, key)
			continue
		}
		counts[record.Namespace]++
	}
	namespaces := make([]StoreNamespaceSummary, 0, len(counts))
	for namespace, count := range counts {
		namespaces = append(namespaces, StoreNamespaceSummary{Name: namespace, RecordCount: count})
	}
	sort.Slice(namespaces, func(i, j int) bool { return namespaces[i].Name < namespaces[j].Name })
	return namespaces
}

// StoreWatch returns records in a namespace with an optional key prefix.
func (s *Service) StoreWatch(namespace, prefix string) []StoreRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	ns := defaultIfBlank(namespace, "default")
	prefix = strings.TrimSpace(prefix)
	out := make([]StoreRecord, 0)
	now := s.now()
	for key, record := range s.store {
		if storeRecordExpired(record, now) {
			delete(s.store, key)
			continue
		}
		if record.Namespace != ns {
			continue
		}
		if prefix != "" && !strings.HasPrefix(record.Key, prefix) {
			continue
		}
		out = append(out, record)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// StoreBind binds an existing record to a device:scenario selector.
func (s *Service) StoreBind(namespace, key, binding string) (StoreRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	storeKey := defaultIfBlank(namespace, "default") + ":" + strings.TrimSpace(key)
	record, ok := s.store[storeKey]
	if !ok {
		return StoreRecord{}, false
	}
	if storeRecordExpired(record, s.now()) {
		delete(s.store, storeKey)
		return StoreRecord{}, false
	}
	record.Binding = strings.TrimSpace(binding)
	record.UpdatedAt = s.now()
	s.store[storeKey] = record
	s.appendRecentLocked("store", storeKey+" bound")
	return record, true
}

func storeRecordExpired(record StoreRecord, now time.Time) bool {
	return record.ExpiresAt != nil && !record.ExpiresAt.After(now)
}
