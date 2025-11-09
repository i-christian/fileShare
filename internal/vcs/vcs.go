package vcs

import (
	"fmt"
	"runtime/debug"
)

func Version() string {
	bi, ok := debug.ReadBuildInfo()
	if ok {
		fmt.Println(bi.Main.Version)
		return bi.Main.Version
	}

	return ""
}
