package main

import (
	"fmt"
	"strings"
)

// version, commit, and buildDate are set at link time with:
//
//	-ldflags "-X main.version=v1.2.3 -X main.commit=abc1234 -X main.buildDate=2026-04-06"
var (
	version   = "dev"
	commit    = ""
	buildDate = ""
)

func printVersion() {
	var b strings.Builder
	b.WriteString("m2a ")
	b.WriteString(version)
	if commit != "" {
		fmt.Fprintf(&b, " (%s", commit)
		if buildDate != "" {
			fmt.Fprintf(&b, ", %s", buildDate)
		}
		b.WriteString(")")
	} else if buildDate != "" {
		fmt.Fprintf(&b, " (%s)", buildDate)
	}
	fmt.Println(b.String())
}
