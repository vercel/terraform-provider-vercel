package vercel

import "strings"

// splitInto2Or3 is a helper function for splitting an import ID into the corresponding parts.
// It also validates whether the ID is in a correct format.
func splitInto2Or3(id string) (teamID, firstID, secondID string, ok bool) {
	attributes := strings.Split(id, "/")
	if len(attributes) == 2 {
		return "", attributes[0], attributes[1], true
	}
	if len(attributes) == 3 {
		return attributes[0], attributes[1], attributes[2], true
	}
	return "", "", "", false
}

// splitInto1Or2 is a helper function for splitting an import ID into the corresponding parts.
// It also validates whether the ID is in a correct format.
func splitInto1Or2(id string) (teamID, firstID string, ok bool) {
	if strings.Contains(id, "/") {
		attributes := strings.Split(id, "/")
		if len(attributes) != 2 {
			return "", "", false
		}
		return attributes[0], attributes[1], true
	}
	return "", id, true
}
