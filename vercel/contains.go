package vercel

func contains[T comparable](items []T, i T) bool {
	for _, j := range items {
		if j == i {
			return true
		}
	}
	return false
}
