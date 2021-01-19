package strings

func AnyOf(testString string, variants ...string) bool {
	for _, s := range variants {
		if testString == s {
			return true
		}
	}
	return false
}
