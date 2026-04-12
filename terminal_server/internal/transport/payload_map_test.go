package transport

import "testing"

func TestEncodeDataMapDeterministic(t *testing.T) {
	data := map[string]string{
		"z": "3",
		"a": "1",
		"m": "2",
	}
	entries := EncodeDataMap(data)
	if len(entries) != 3 {
		t.Fatalf("len(entries) = %d, want 3", len(entries))
	}
	if entries[0].Key != "a" || entries[1].Key != "m" || entries[2].Key != "z" {
		t.Fatalf("entry order = %+v", entries)
	}
}

func TestDecodeDataEntries(t *testing.T) {
	decoded := DecodeDataEntries([]DataEntry{
		{Key: "a", Value: "1"},
		{Key: "b", Value: "2"},
		{Key: "a", Value: "3"},
	})
	if decoded["a"] != "3" || decoded["b"] != "2" {
		t.Fatalf("decoded = %+v", decoded)
	}
}
