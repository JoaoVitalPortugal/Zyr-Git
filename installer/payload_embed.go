//go:build installerbuild

package main

import _ "embed"

//go:embed payload/zyr.exe
var launcherPayload []byte

//go:embed payload/zyr-git-commit.exe
var gitComponentPayload []byte
