package compiler_cli

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ToNumbers converts a `string` version into an array of numbers
// @example
// ToNumbers('2.0.1'); // returns [2, 0, 1]
func ToNumbers(value string) []int {
	// Drop any suffixes starting with `-` so that versions like `1.2.3-rc.5` are treated as `1.2.3`.
	suffixIndex := strings.LastIndex(value, "-")
	var sliced string
	if suffixIndex == -1 {
		sliced = value
	} else {
		sliced = value[:suffixIndex]
	}

	segments := strings.Split(sliced, ".")
	numbers := make([]int, len(segments))
	for i, segment := range segments {
		parsed, err := strconv.Atoi(segment)
		if err != nil {
			panic(fmt.Errorf("Unable to parse version string %s.", value))
		}
		numbers[i] = parsed
	}
	return numbers
}

// CompareNumbers compares two arrays of positive numbers with lexicographical order in mind.
//
// However - unlike lexicographical order - for arrays of different length we consider:
// [1, 2, 3] = [1, 2, 3, 0] instead of [1, 2, 3] < [1, 2, 3, 0]
//
// @param a The 'left hand' array in the comparison test
// @param b The 'right hand' in the comparison test
// @returns {-1|0|1} The comparison result: 1 if a is greater, -1 if b is greater, 0 is the two
// arrays are equals
func CompareNumbers(a []int, b []int) int {
	maxLen := int(math.Max(float64(len(a)), float64(len(b))))
	minLen := int(math.Min(float64(len(a)), float64(len(b))))

	for i := 0; i < minLen; i++ {
		if a[i] > b[i] {
			return 1
		}
		if a[i] < b[i] {
			return -1
		}
	}

	if minLen != maxLen {
		var longestArray []int
		var comparisonResult int
		if len(a) == maxLen {
			longestArray = a
			comparisonResult = 1
		} else {
			longestArray = b
			comparisonResult = -1
		}

		// Check that at least one of the remaining elements is greater than 0 to consider that the two
		// arrays are different (e.g. [1, 0] and [1] are considered the same but not [1, 0, 1] and [1])
		for i := minLen; i < maxLen; i++ {
			if longestArray[i] > 0 {
				return comparisonResult
			}
		}
	}

	return 0
}

// IsVersionBetween checks if a TypeScript version is:
// - greater or equal than the provided `low` version,
// - lower or equal than an optional `high` version.
//
// @param version The TypeScript version
// @param low The minimum version
// @param high The maximum version
func IsVersionBetween(version string, low string, high *string) bool {
	tsNumbers := ToNumbers(version)
	if high != nil {
		return CompareNumbers(ToNumbers(low), tsNumbers) <= 0 &&
			CompareNumbers(ToNumbers(*high), tsNumbers) >= 0
	}
	return CompareNumbers(ToNumbers(low), tsNumbers) <= 0
}

// CompareVersions compares two versions
//
// @param v1 The 'left hand' version in the comparison test
// @param v2 The 'right hand' version in the comparison test
// @returns {-1|0|1} The comparison result: 1 if v1 is greater, -1 if v2 is greater, 0 is the two
// versions are equals
func CompareVersions(v1 string, v2 string) int {
	return CompareNumbers(ToNumbers(v1), ToNumbers(v2))
}
