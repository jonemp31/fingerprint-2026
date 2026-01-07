package handlers

// truncateURL truncates a URL to 60 characters for logging
func truncateURL(url string) string {
	if len(url) > 60 {
		return url[:57] + "..."
	}
	return url
}
