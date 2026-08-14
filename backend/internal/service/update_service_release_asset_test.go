//go:build unit

package service

import (
	"context"
	"testing"
	"time"
)

type noopUpdateCache struct{}

func (noopUpdateCache) GetUpdateInfo(context.Context) (string, error) {
	return "", nil
}

func (noopUpdateCache) SetUpdateInfo(context.Context, string, time.Duration) error {
	return nil
}

func TestUpdateServiceCompatibleReleaseAsset(t *testing.T) {
	svc := NewUpdateService(noopUpdateCache{}, nil, "99.12.10", "release")

	tests := []struct {
		name string
		want bool
	}{
		{name: "sub2api-linux-amd64", want: true},
		{name: "sub2api-linux-amd64.sha256", want: false},
		{name: "sub2api_99.12.11_linux_amd64.tar.gz", want: true},
		{name: "checksums.txt", want: false},
		{name: "sub2api_99.12.11_linux_arm64.tar.gz", want: false},
	}

	for _, tt := range tests {
		if got := svc.isCompatibleReleaseAsset(tt.name); got != tt.want {
			t.Fatalf("isCompatibleReleaseAsset(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}
