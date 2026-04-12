package config

import "testing"

func TestNewConfig(t *testing.T) {
	cfg := NewConfig()
	if cfg == nil {
		t.Error("Expected a config, got nil")
	}
}
