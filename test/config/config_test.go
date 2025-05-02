package config

import (
	"testing"

	"github.com/moritzrinow/regmaid/internal/regmaid"
)

func TestLoadConfig(t *testing.T) {
	cfg, err := regmaid.LoadConfig("../regmaid.yaml")
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	t.Logf("loaded config: %v", cfg)
}
