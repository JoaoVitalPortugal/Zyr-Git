//go:build installerbuild

package main

import _ "embed"

//go:embed payload/zyr.exe
var launcherPayload []byte

//go:embed payload/zyr-git.exe
var gitComponentPayload []byte

//go:embed payload/zyr-git-dashboard.exe
var dashboardPayload []byte

//go:embed assets/zyr-git.ico
var iconPayload []byte
