package vercel

func contains(items []string, i string) bool {
	for _, j := range items {
		if j == i {
			return true
		}
	}
	return false
}
