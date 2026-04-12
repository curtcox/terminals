package ui

import "testing"

func TestValidateSuccess(t *testing.T) {
	d := New("stack", nil, New("text", map[string]string{"value": "hello"}))
	if err := Validate(d); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateFailure(t *testing.T) {
	d := New("weird_custom_widget", nil)
	if err := Validate(d); err == nil {
		t.Fatalf("Validate() expected error for unsupported type")
	}
}
