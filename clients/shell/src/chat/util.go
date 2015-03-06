package main

import (
	"strings"

	"v.io/v23/security"
)

// Calculate the longest common prefix from an array of strings.
func longestCommonPrefix(strings []string) string {
	if len(strings) == 0 {
		return ""
	}
	first := strings[0]
	strings = strings[1:]
	for i := range first {
		for _, s := range strings {
			if len(s) <= i || s[i] != first[i] {
				return first[:i]
			}
		}
	}
	return first
}

// Note, shortName and firstShortName are duplicated between JS and Go.
func shortName(fullName string) string {
	// Split into components and see if any is an email address. A very
	// sophisticated technique is used to determine if the component is an email
	// address: presence of an "@" character.
	parts := strings.Split(string(fullName), security.ChainSeparator)
	for _, p := range parts {
		if strings.Count(p, "@") == 1 {
			return p
		}
	}

	// If no email address is found, use the fullName. Useful for testing.
	return fullName
}

func firstShortName(blessings []string) string {
	if len(blessings) == 0 {
		return "unknown"
	}
	for _, blessing := range blessings {
		if sn := shortName(blessing); sn != "" {
			return sn
		}
	}
	return string(blessings[0])
}

// uniqStrings takes a *sorted* slice of strings and returns a slice with all
// duplicate entries removed.
func uniqStrings(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}

	out := make([]string, len(in))
	out[0] = in[0]

	o := 0

	for i := 1; i < len(in); i++ {
		if out[o] != in[i] {
			o++
			out[o] = in[i]
		}
	}

	return out[:o+1]
}
