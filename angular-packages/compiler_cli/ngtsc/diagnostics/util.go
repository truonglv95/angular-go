package diagnostics

import (
	"regexp"
	"strconv"
)

var errorCodeMatcher = regexp.MustCompile(`(\x1b\[\d+m ?)TS-99(\d+: ?\x1b\[\d+m)`)

func ReplaceTsWithNgInErrors(errors string) string {
	return errorCodeMatcher.ReplaceAllString(errors, "${1}NG${2}")
}

func NgErrorCode(code ErrorCode) int {
	// Equivalent to parseInt('-99' + code)
	val, err := strconv.Atoi("-99" + strconv.Itoa(int(code)))
	if err != nil {
		return int(code) // fallback
	}
	return val
}
