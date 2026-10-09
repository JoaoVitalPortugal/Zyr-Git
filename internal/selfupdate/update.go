package selfupdate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	releasesURL = "https://api.github.com/repos/JoaoVitalPortugal/Zyr-Git/releases/latest"
	assetName   = "ZyrGit-Setup.exe"
	maxSize     = 200 << 20
)

type Release struct {
	Version     string `json:"version"`
	PublishedAt string `json:"publishedAt"`
	Notes       string `json:"notes"`
	AssetURL    string `json:"assetUrl"`
	Digest      string `json:"digest"`
}

type cacheFile struct {
	CheckedAt time.Time `json:"checkedAt"`
	Release   Release   `json:"release"`
}

type Client struct {
	HTTP         *http.Client
	DownloadHTTP *http.Client
	CachePath    string
	URL          string
}

func New() *Client {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		base = os.TempDir()
	}
	return &Client{
		HTTP:         &http.Client{Timeout: 3 * time.Second},
		DownloadHTTP: &http.Client{Timeout: 5 * time.Minute},
		CachePath:    filepath.Join(base, "Zyr Git", "update-cache.json"),
		URL:          releasesURL,
	}
}

func (c *Client) Check(ctx context.Context, current string) (Release, bool, error) {
	var cached cacheFile
	if data, err := os.ReadFile(c.CachePath); err == nil {
		_ = json.Unmarshal(data, &cached)
	}
	if time.Since(cached.CheckedAt) >= 0 && time.Since(cached.CheckedAt) < 10*time.Minute {
		return cached.Release, newer(cached.Release.Version, current), nil
	}

	release, err := c.fetch(ctx)
	if err != nil {
		if cached.Release.Version != "" {
			return cached.Release, false, nil
		}
		return Release{}, false, err
	}
	data, _ := json.Marshal(cacheFile{CheckedAt: time.Now(), Release: release})
	if err := os.MkdirAll(filepath.Dir(c.CachePath), 0o700); err == nil {
		_ = os.WriteFile(c.CachePath, data, 0o600)
	}
	return release, newer(release.Version, current), nil
}

func (c *Client) fetch(ctx context.Context) (Release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.URL, nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Zyr-Git-Updater")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return Release{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return Release{}, nil // No release has been published yet.
	}
	if resp.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("GitHub retornou HTTP %d", resp.StatusCode)
	}
	var payload struct {
		TagName     string `json:"tag_name"`
		PublishedAt string `json:"published_at"`
		Body        string `json:"body"`
		Assets      []struct {
			Name   string `json:"name"`
			URL    string `json:"browser_download_url"`
			Digest string `json:"digest"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&payload); err != nil {
		return Release{}, err
	}
	version := strings.TrimPrefix(payload.TagName, "v")
	if !validVersion(version) {
		return Release{}, fmt.Errorf("versão de release inválida: %q", payload.TagName)
	}
	for _, asset := range payload.Assets {
		if asset.Name == assetName {
			parsed, err := url.Parse(asset.URL)
			if err != nil || parsed.Scheme != "https" || !strings.EqualFold(parsed.Host, "github.com") || !strings.HasPrefix(parsed.Path, "/JoaoVitalPortugal/Zyr-Git/releases/download/") {
				return Release{}, errors.New("endereço do instalador inválido")
			}
			if len(asset.Digest) != len("sha256:")+64 || !strings.HasPrefix(asset.Digest, "sha256:") {
				return Release{}, errors.New("release sem digest SHA-256 válido")
			}
			if _, err := hex.DecodeString(strings.TrimPrefix(asset.Digest, "sha256:")); err != nil {
				return Release{}, errors.New("digest SHA-256 inválido")
			}
			return Release{Version: version, PublishedAt: payload.PublishedAt, Notes: payload.Body, AssetURL: asset.URL, Digest: asset.Digest}, nil
		}
	}
	return Release{}, fmt.Errorf("release %s sem %s", payload.TagName, assetName)
}

func newer(candidate, current string) bool {
	a, okA := parts(candidate)
	b, okB := parts(current)
	if !okA || !okB {
		return false
	}
	for index := range a {
		if a[index] != b[index] {
			return a[index] > b[index]
		}
	}
	return false
}

func validVersion(value string) bool {
	_, ok := parts(value)
	return ok
}

func parts(value string) ([3]int, bool) {
	var result [3]int
	fields := strings.Split(value, ".")
	if len(fields) != 3 {
		return result, false
	}
	for index, field := range fields {
		if field == "" || strings.Trim(field, "0123456789") != "" {
			return result, false
		}
		number, err := strconv.Atoi(field)
		if err != nil {
			return result, false
		}
		result[index] = number
	}
	return result, true
}

func (c *Client) Install(ctx context.Context, release Release, progress func(stage string, percent int)) error {
	if !validVersion(release.Version) || !validAsset(release.AssetURL, release.Digest) {
		return errors.New("release inválida")
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	versionDir := filepath.Dir(self)
	versionsDir := filepath.Dir(versionDir)
	if filepath.Base(versionsDir) != "versions" {
		return errors.New("o dashboard não está em uma instalação do Zyr Git")
	}
	installDir := filepath.Dir(versionsDir)
	sharedHome := os.Getenv("ZYR_CLI_HOME")
	if sharedHome == "" {
		sharedHome = filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Zyr CLI")
	}
	return c.installAt(ctx, release, installDir, sharedHome, progress)
}

func (c *Client) installAt(ctx context.Context, release Release, installDir, sharedHome string, progress func(stage string, percent int)) error {
	if !validVersion(release.Version) || !validAsset(release.AssetURL, release.Digest) {
		return errors.New("release inválida")
	}
	lockPath := filepath.Join(installDir, "update.lock")
	lock, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		if info, statErr := os.Stat(lockPath); statErr == nil && time.Since(info.ModTime()) > 15*time.Minute {
			_ = os.Remove(lockPath)
			lock, err = os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		}
	}
	if err != nil {
		return fmt.Errorf("outra atualização já está em andamento: %w", err)
	}
	lock.Close()
	defer os.Remove(lockPath)
	progress("Baixando a atualização", 0)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, release.AssetURL, nil)
	if err != nil {
		return err
	}
	downloadHTTP := c.DownloadHTTP
	if downloadHTTP == nil {
		downloadHTTP = &http.Client{Timeout: 5 * time.Minute}
	}
	response, err := downloadHTTP.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || response.ContentLength > maxSize {
		return fmt.Errorf("download rejeitado: HTTP %d", response.StatusCode)
	}
	file, err := os.CreateTemp("", "zyr-git-update-*.exe")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	hash := sha256.New()
	buffer := make([]byte, 64<<10)
	var copied int64
	for {
		count, readErr := response.Body.Read(buffer)
		if count > 0 {
			copied += int64(count)
			if copied > maxSize {
				return errors.New("instalador excedeu o tamanho máximo")
			}
			if _, err := file.Write(buffer[:count]); err != nil {
				return err
			}
			_, _ = hash.Write(buffer[:count])
			if response.ContentLength > 0 {
				progress("Baixando a atualização", int(copied*100/response.ContentLength))
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	progress("Verificando o instalador", -1)
	if "sha256:"+hex.EncodeToString(hash.Sum(nil)) != strings.ToLower(release.Digest) {
		return errors.New("o instalador baixado não passou na verificação SHA-256")
	}
	if err := file.Close(); err != nil {
		return err
	}
	progress("Instalando a nova versão", -1)
	cmd := exec.CommandContext(ctx, file.Name(), "--silent", "--auto-update", "--install-dir", installDir, "--shared-home", sharedHome)
	configureHidden(cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("instalação falhou: %w: %s", err, strings.TrimSpace(string(output)))
	}
	manifest, err := os.ReadFile(filepath.Join(sharedHome, "components", "git.json"))
	if err != nil {
		return fmt.Errorf("não foi possível confirmar a versão instalada: %w", err)
	}
	var installed struct {
		Version string `json:"version"`
	}
	if json.Unmarshal(manifest, &installed) != nil || installed.Version != release.Version {
		return errors.New("o instalador terminou sem ativar a versão esperada")
	}
	progress("Atualização concluída", 100)
	return nil
}

func validAsset(assetURL, digest string) bool {
	parsed, err := url.Parse(assetURL)
	if err != nil || parsed.Scheme != "https" || !strings.EqualFold(parsed.Host, "github.com") || !strings.HasPrefix(parsed.Path, "/JoaoVitalPortugal/Zyr-Git/releases/download/") {
		return false
	}
	if len(digest) != len("sha256:")+64 || !strings.HasPrefix(digest, "sha256:") {
		return false
	}
	_, err = hex.DecodeString(strings.TrimPrefix(digest, "sha256:"))
	return err == nil
}
