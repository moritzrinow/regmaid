package client

import (
	"context"
	"testing"

	"github.com/moritzrinow/regmaid/internal/regmaid"
	"github.com/regclient/regclient/types/ref"
)

func TestClient(t *testing.T) {
	cfg, err := regmaid.LoadConfig("../regmaid.yaml")
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	client := regmaid.NewRegistryClient(cfg)

	ref, err := ref.New("example.com/repo")

	if err != nil {
		t.Fatalf("error creating ref: %v", err)

		return
	}

	tagLists, err := client.TagList(context.Background(), ref)

	if err != nil {
		t.Fatalf("error listing tags: %v", err)
	}

	tags, _ := tagLists.GetTags()

	for _, tag := range tags {
		t.Logf(tag)
	}
}
