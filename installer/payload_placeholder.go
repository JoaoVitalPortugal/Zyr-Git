//go:build !installerbuild

package main

// The release build replaces these with freshly compiled payloads.
var launcherPayload []byte
var gitComponentPayload []byte
