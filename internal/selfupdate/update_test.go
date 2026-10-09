package selfupdate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestVersionComparisonOnlyAcceptsNewerStableRelease(t *testing.T) {
	for _, test := range []struct {
		candidate string
		current   string
		want      bool
	}{
		{"0.6.1", "0.6.0", true},
		{"0.6.0", "0.6.0", false},
		{"0.5.9", "0.6.0", false},
		{"0.7.0-beta", "0.6.0", false},
	} {
		if got := newer(test.candidate, test.current); got != test.want {
			t.Errorf("newer(%q, %q) = %v, want %v", test.candidate, test.current, got, test.want)
		}
	}
}

type releaseTransport struct{ server *url.URL }

func (transport releaseTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	copy := request.Clone(request.Context())
	copy.URL.Scheme = transport.server.Scheme
	copy.URL.Host = transport.server.Host
	copy.Host = transport.server.Host
	return http.DefaultTransport.RoundTrip(copy)
}

func TestInstallerVerifiesDigestAndActivatesExpectedVersion(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "fake-installer.go")
	code := `package main
import ("os"; "path/filepath")
func main() {
  home := ""
  for i, arg := range os.Args { if arg == "--shared-home" && i+1 < len(os.Args) { home = os.Args[i+1] } }
  if home == "" { os.Exit(2) }
  directory := filepath.Join(home, "components")
  if os.MkdirAll(directory, 0755) != nil { os.Exit(3) }
  if os.WriteFile(filepath.Join(directory, "git.json"), []byte("{\"version\":\"0.7.0\"}"), 0600) != nil { os.Exit(4) }
}`
	if err := os.WriteFile(source, []byte(code), 0o600); err != nil {
		t.Fatal(err)
	}
	goName := "go"
	if runtime.GOOS == "windows" {
		goName += ".exe"
	}
	binary := filepath.Join(root, "fake-installer.exe")
	build := exec.Command(filepath.Join(runtime.GOROOT(), "bin", goName), "build", "-o", binary, source)
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("fake installer build failed: %v\n%s", err, output)
	}
	content, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Length", fmt.Sprint(len(content)))
		_, _ = writer.Write(content)
	}))
	defer server.Close()
	serverURL, _ := url.Parse(server.URL)
	client := &Client{DownloadHTTP: &http.Client{Transport: releaseTransport{server: serverURL}}}
	release := Release{
		Version:  "0.7.0",
		AssetURL: "https://github.com/JoaoVitalPortugal/Zyr-Git/releases/download/v0.7.0/ZyrGit-Setup.exe",
		Digest:   "sha256:" + hex.EncodeToString(sha256.New().Sum(nil)),
	}
	installDir := filepath.Join(root, "install")
	sharedHome := filepath.Join(root, "shared")
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := client.installAt(context.Background(), release, installDir, sharedHome, func(string, int) {}); err == nil {
		t.Fatal("incorrect digest was accepted")
	}
	if _, err := os.Stat(filepath.Join(sharedHome, "components", "git.json")); !os.IsNotExist(err) {
		t.Fatal("installer ran before digest verification")
	}
	sum := sha256.Sum256(content)
	release.Digest = "sha256:" + hex.EncodeToString(sum[:])
	var stages []string
	if err := client.installAt(context.Background(), release, installDir, sharedHome, func(stage string, percent int) { stages = append(stages, stage) }); err != nil {
		t.Fatal(err)
	}
	if len(stages) == 0 || stages[len(stages)-1] != "Atualização concluída" {
		t.Fatalf("update did not finish: %v", stages)
	}
	manifest, err := os.ReadFile(filepath.Join(sharedHome, "components", "git.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(manifest), "0.7.0") {
		t.Fatalf("wrong installed version: %s", manifest)
	}
}

func TestCheckReadsReleaseAndUsesCache(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tag_name":     "v0.7.0",
			"published_at": "2026-10-09T12:00:00Z",
			"assets": []map[string]string{{
				"name":                 assetName,
				"browser_download_url": "https://github.com/JoaoVitalPortugal/Zyr-Git/releases/download/v0.7.0/ZyrGit-Setup.exe",
				"digest":               "sha256:" + strings.Repeat("a", 64),
			}},
		})
	}))
	defer server.Close()
	client := &Client{HTTP: server.Client(), URL: server.URL, CachePath: filepath.Join(t.TempDir(), "cache.json")}
	for range 2 {
		release, available, err := client.Check(context.Background(), "0.6.0")
		if err != nil || !available || release.Version != "0.7.0" {
			t.Fatalf("unexpected release: %+v %v %v", release, available, err)
		}
	}
	if requests != 1 {
		t.Fatalf("expected one network request, got %d", requests)
	}
}

func TestCheckWithoutReleaseDoesNotPromptUpdate(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	client := &Client{HTTP: server.Client(), URL: server.URL, CachePath: filepath.Join(t.TempDir(), "cache.json")}
	_, available, err := client.Check(context.Background(), "0.6.0")
	if err != nil || available {
		t.Fatalf("no release should not update: %v %v", available, err)
	}
}

func TestReleaseRejectsUntrustedInstaller(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tag_name": "v0.7.0",
			"assets":   []map[string]string{{"name": assetName, "browser_download_url": "https://example.com/installer.exe", "digest": "sha256:" + strings.Repeat("a", 64)}},
		})
	}))
	defer server.Close()
	client := &Client{HTTP: server.Client(), URL: server.URL, CachePath: filepath.Join(t.TempDir(), "cache.json")}
	if _, _, err := client.Check(context.Background(), "0.6.0"); err == nil {
		t.Fatal("untrusted installer URL was accepted")
	}
}
