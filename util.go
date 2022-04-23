package main

// A way to determine if a particular string is in a particular slice.
func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// A constant string slice of all roles that are admin and higher.
// Currently "admin" and "owner"
func adminRoles() []string {
	return []string{"admin", "owner"}
}