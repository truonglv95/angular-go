package diagnostics

import "strings"

// TODO: import {VERSION} from '@angular/compiler';

func getDocPageBaseUrl(versionFull string, versionMajor string) string {
	isPreRelease := strings.Contains(versionFull, "-next") || strings.Contains(versionFull, "-rc") || versionFull == "0.0.0-PLACEHOLDER"
	prefix := "v" + versionMajor
	if isPreRelease {
		prefix = "next"
	}
	return "https://" + prefix + ".angular.dev"
}

// In Go, variables are evaluated at init or we can use functions.
// We'll export functions that take version to remain pure, or just hardcode for typical usage.
// Since we don't have the exact VERSION package port right now, we can leave this as a function or placeholder.
