//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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

type updateServiceCacheStub struct {
	data string
}

func (s *updateServiceCacheStub) GetUpdateInfo(context.Context) (string, error) {
	if s.data == "" {
		return "", errors.New("cache miss")
	}
	return s.data, nil
}

func (s *updateServiceCacheStub) SetUpdateInfo(_ context.Context, data string, _ time.Duration) error {
	s.data = data
	return nil
}

type updateServiceGitHubClientStub struct {
	release *GitHubRelease
}

func (s *updateServiceGitHubClientStub) FetchLatestRelease(context.Context, string) (*GitHubRelease, error) {
	return s.release, nil
}

func (s *updateServiceGitHubClientStub) DownloadFile(context.Context, string, string, int64) error {
	panic("DownloadFile should not be called when no update is available")
}

func (s *updateServiceGitHubClientStub) FetchChecksumFile(context.Context, string) ([]byte, error) {
	panic("FetchChecksumFile should not be called when no update is available")
}

func TestUpdateServicePerformUpdateNoUpdateReturnsSentinel(t *testing.T) {
	svc := NewUpdateService(
		&updateServiceCacheStub{},
		&updateServiceGitHubClientStub{
			release: &GitHubRelease{
				TagName: "v0.1.132",
				Name:    "v0.1.132",
			},
		},
		"0.1.132",
		"release",
	)

	err := svc.PerformUpdate(context.Background())

	require.Error(t, err)
	require.True(t, errors.Is(err, ErrNoUpdateAvailable))
	require.ErrorIs(t, err, ErrNoUpdateAvailable)
}
