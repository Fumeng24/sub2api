package service

import (
	"fmt"
	"runtime"
	"strings"
)

func (s *UpdateService) selectReleaseAssetsCustom(releaseAssets []Asset) (string, string) {
	downloadURL := ""
	selectedAssetName := ""
	for _, asset := range releaseAssets {
		if s.isCompatibleReleaseAsset(asset.Name) {
			downloadURL = asset.DownloadURL
			selectedAssetName = asset.Name
			break
		}
	}

	checksumURL := ""
	for _, asset := range releaseAssets {
		if selectedAssetName != "" && asset.Name == selectedAssetName+".sha256" {
			checksumURL = asset.DownloadURL
			break
		}
		if checksumURL == "" && asset.Name == "checksums.txt" {
			checksumURL = asset.DownloadURL
		}
	}
	return downloadURL, checksumURL
}

func (s *UpdateService) isCompatibleReleaseAsset(name string) bool {
	if isUpdateChecksumAsset(name) {
		return false
	}

	assetName := strings.ToLower(name)
	for _, pattern := range s.getPlatformAssetNamePatterns() {
		if strings.Contains(assetName, pattern) {
			return true
		}
	}
	return false
}

func (s *UpdateService) getPlatformAssetNamePatterns() []string {
	osName := runtime.GOOS
	arch := runtime.GOARCH
	return []string{
		fmt.Sprintf("%s_%s", osName, arch),
		fmt.Sprintf("%s-%s", osName, arch),
	}
}

func isUpdateChecksumAsset(name string) bool {
	assetName := strings.ToLower(name)
	return assetName == "checksums.txt" ||
		strings.HasSuffix(assetName, ".sha256") ||
		strings.HasSuffix(assetName, ".sha256sum")
}
