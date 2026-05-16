package launcher

import "testing"

func TestSelectPlayableAssetPrefersCurrentPlatformBinary(t *testing.T) {
	release := Release{
		Assets: []ReleaseAsset{
			{Name: "notes.txt"},
			{Name: "poblation_windows_amd64.exe", BrowserDownloadURL: "https://example.invalid/poblation.exe"},
		},
	}

	asset, ok := release.SelectPlayableAsset("windows", "amd64")
	if !ok {
		t.Fatalf("expected playable asset")
	}
	if asset.BrowserDownloadURL == "" {
		t.Fatalf("expected browser download url")
	}
}
