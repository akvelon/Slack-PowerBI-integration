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

//SliceStringDifference returns a new array by taking two arrays returning the first one without rows from the second one.
func GetArrayDifference(firstArray []string, secondArray []string) []string {
	arrayDifference := make([]string, len(firstArray))
	copy(arrayDifference, firstArray)
	for i := range secondArray {
		for j := range arrayDifference {
			if arrayDifference[j] == secondArray[i] {
				arrayDifference = append(arrayDifference[:j], arrayDifference[j+1:]...)
				break
			}
		}
	}

	return arrayDifference
}
