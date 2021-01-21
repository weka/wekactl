package strings

import "strings"

func AnyOf(testString string, variants ...string) bool {
	for _, s := range variants {
		if testString == s {
			return true
		}
	}
	return false
}

func AnyOfSubstr(testString string, variants ...string) bool {
	for _, s := range variants {
		if strings.Contains(testString, s) {
			return true
		}
	}
	return false
}
