package main

import (
	"strings"
)

// isReservedPositional reports words that must be passed as flags, not as the lone URL argument.
func isReservedPositional(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug", "raw", "version":
		return true
	default:
		return false
	}
}

// reservedFlagHint returns the flag form for a reserved word (e.g. "-debug").
func reservedFlagHint(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return "-debug"
	case "raw":
		return "-raw"
	case "version":
		return "-version or: m2a version"
	default:
		return ""
	}
}

// flagsThatTakeNextArg are long-form flag names (after stripping leading dashes) whose value
// may be passed as the following argv token (e.g. -base https://x).
var flagsThatTakeNextArg = map[string]struct{}{
	"base":       {},
	"card":       {},
	"card-path":  {},
	"transport":  {},
	"H":          {},
}

// boolFlagNames are flags that never consume a following argv token (values use -name=value).
// Keep in sync with flag.Bool / flag.BoolVar definitions in main.
var boolFlagNames = map[string]struct{}{
	"raw":     {},
	"debug":   {},
	"version": {},
}

// reorderFlagsBeforePositionals moves flag tokens (and their values) before positional
// arguments. This matches common CLI expectations: `m2a http://host -debug` works the
// same as `m2a -debug http://host`. Go's flag package alone does not allow flags after
// the first positional.
func reorderFlagsBeforePositionals(args []string) []string {
	var flags, pos []string
	for i := 0; i < len(args); {
		a := args[i]
		if a == "--" {
			pos = append(pos, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(a, "-") {
			pos = append(pos, a)
			i++
			continue
		}
		name, _, hasEq := splitFlagArg(a)
		if hasEq {
			flags = append(flags, a)
			i++
			continue
		}
		if _, isBool := boolFlagNames[name]; isBool {
			flags = append(flags, a)
			i++
			continue
		}
		if _, ok := flagsThatTakeNextArg[name]; ok {
			flags = append(flags, a)
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				flags = append(flags, args[i+1])
				i += 2
				continue
			}
			i++
			continue
		}
		flags = append(flags, a)
		i++
	}
	return append(flags, pos...)
}

// splitFlagArg returns the flag name and whether the value is in the same argv element (-f=x).
func splitFlagArg(s string) (name string, value string, hasEquals bool) {
	s = strings.TrimPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	if i := strings.IndexByte(s, '='); i >= 0 {
		return s[:i], s[i+1:], true
	}
	return s, "", false
}
