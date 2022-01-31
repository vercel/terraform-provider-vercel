package vercel

import "strings"

// trimFilePath removes any upward directory navigations from the start of a file path.
// This is useful as Vercel doesn't allow a root directory that navigates upwards.
// But if we trim all file paths upwards, and we also trim the root directory, then
// all is good.
func trimFilePath(path string) string {
	for strings.HasPrefix(path, "../") {
		path = strings.TrimPrefix(path, "../")
	}
	if path == "" {
		return "."
	}

	return path
}
