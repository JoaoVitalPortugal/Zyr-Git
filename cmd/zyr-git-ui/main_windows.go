package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/JoaoVitalPortugal/zyr-git/internal/selfupdate"
)

//go:embed dashboard.ps1
var dashboardScript string

//go:embed progress.ps1
var progressScript string

var version = "dev"

type progressState struct {
	Stage   string `json:"stage"`
	Percent int    `json:"percent"`
	Done    bool   `json:"done"`
	Error   bool   `json:"error"`
	Version string `json:"version"`
}

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--zyr-ui-protocol" {
		fmt.Println("zyr-git-ui/1")
		return
	}
	if len(os.Args) == 3 && os.Args[1] == "--shared-home" {
		_ = os.Setenv("ZYR_CLI_HOME", os.Args[2])
	}
	client := selfupdate.New()
	release, available, err := client.Check(context.Background(), version)
	if err != nil {
		if len(os.Args) > 1 && os.Args[1] == "--check-and-update" {
			os.Exit(1)
		}
	}
	if available {
		if err := update(client, release); err != nil {
			if len(os.Args) > 1 && os.Args[1] == "--check-and-update" {
				os.Exit(1)
			}
			showDashboard(release)
			return
		}
		os.Exit(10)
	}
	if len(os.Args) > 1 && os.Args[1] == "--check-and-update" {
		return
	}
	showDashboard(release)
}

func update(client *selfupdate.Client, release selfupdate.Release) error {
	stateFile, err := os.CreateTemp("", "zyr-git-progress-*.json")
	if err != nil {
		return err
	}
	path := stateFile.Name()
	stateFile.Close()
	defer os.Remove(path)
	writeState := func(state progressState) {
		data, _ := json.Marshal(state)
		_ = os.WriteFile(path, data, 0o600)
	}
	writeState(progressState{Stage: "Preparando a atualização", Percent: -1, Version: release.Version})
	window, cleanup, err := powershell(progressScript, "ZYR_GIT_PROGRESS_FILE="+path)
	if err != nil {
		return err
	}
	defer cleanup()
	if err := window.Start(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	installErr := client.Install(ctx, release, func(stage string, percent int) {
		writeState(progressState{Stage: stage, Percent: percent, Version: release.Version})
	})
	if installErr != nil {
		writeState(progressState{Stage: installErr.Error(), Percent: -1, Done: true, Error: true, Version: release.Version})
	} else {
		writeState(progressState{Stage: "Atualização concluída", Percent: 100, Done: true, Version: release.Version})
	}
	_ = window.Wait()
	return installErr
}

func showDashboard(release selfupdate.Release) {
	dataFile, err := os.CreateTemp("", "zyr-git-dashboard-*.json")
	if err != nil {
		return
	}
	path := dataFile.Name()
	dataFile.Close()
	defer os.Remove(path)
	installed := ""
	home := os.Getenv("ZYR_CLI_HOME")
	if home == "" {
		home = filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Zyr CLI")
	}
	if home != "" {
		manifest := filepath.Join(home, "components", "git.json")
		if data, err := os.ReadFile(manifest); err == nil {
			var value struct {
				InstalledAt string `json:"installedAt"`
			}
			if json.Unmarshal(data, &value) == nil {
				installed = value.InstalledAt
			}
		}
	}
	data, _ := json.Marshal(map[string]string{
		"version": version, "installedAt": installed,
		"latest": release.Version, "publishedAt": release.PublishedAt,
	})
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return
	}
	window, cleanup, err := powershell(dashboardScript, "ZYR_GIT_DASHBOARD_FILE="+path)
	if err != nil {
		return
	}
	defer cleanup()
	_ = window.Run()
}

func powershell(script string, environment string) (*exec.Cmd, func(), error) {
	file, err := os.CreateTemp("", "zyr-git-ui-*.ps1")
	if err != nil {
		return nil, nil, err
	}
	path := file.Name()
	cleanup := func() { _ = os.Remove(path) }
	// Windows PowerShell 5.1 needs a BOM to read non-ASCII scripts as UTF-8.
	if _, err := file.Write(append([]byte{0xEF, 0xBB, 0xBF}, []byte(script)...)); err != nil {
		file.Close()
		cleanup()
		return nil, nil, err
	}
	if err := file.Close(); err != nil {
		cleanup()
		return nil, nil, err
	}
	cmd := exec.Command("powershell.exe", "-NoLogo", "-NoProfile", "-NonInteractive", "-Sta", "-WindowStyle", "Hidden", "-File", path)
	self, _ := os.Executable()
	icon := filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(self))), "zyr-git.ico")
	cmd.Env = append(os.Environ(), environment, "ZYR_GIT_ICON="+icon)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	if os.Getenv("ZYR_GIT_UI_SMOKE") == "1" {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return cmd, cleanup, nil
}
