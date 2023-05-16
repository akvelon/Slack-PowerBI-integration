package utils

// Contains checks string in an string array
func Contains(s []string, v string) bool {
	for _, e := range s {
		if e == v {
			return true
		}
	}

	return false
}
