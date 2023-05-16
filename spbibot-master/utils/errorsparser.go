package utils

import "strconv"

func AuthorizationError(a string) bool {
	res, err := strconv.Atoi(a[len(a)-3:])
	if err != nil {
		res = 0
	}
	if res == 401 || res == 403 {
		return true
	}
	return false
}
