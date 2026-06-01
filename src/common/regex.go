package common

// Package: common
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import "regexp"

// Pre-compiled regular expressions for input validation.
// These patterns are used across many packages; compiling once
// at init avoids re-parsing on every call.
var (
	AlphanumericRegex         = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	AlphanumericDashRegex     = regexp.MustCompile(`^[a-zA-Z0-9-]+$`)
	AlphanumericDashSemiColon = regexp.MustCompile(`^[a-zA-Z0-9-;:]+$`)
	AlphanumericUnderscore    = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	AlphanumericUnderDashPath = regexp.MustCompile(`^[a-zA-Z0-9_\-/]+$`)
	AlphanumericUnderColon    = regexp.MustCompile(`^[a-zA-Z0-9_:-]+$`)
	AlphanumericDisplayName   = regexp.MustCompile(`^[a-zA-Z0-9#.:_ -]*$`)
)
