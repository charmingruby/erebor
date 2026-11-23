package api

// Response contains command reply text and optional images displayed below it.
// ImageURLs must be publicly accessible HTTP(S) image URLs.
// Images are sent in order, with at most 10 per message.
// An empty response removes the deferred command response.
type Response struct {
	Content   string
	ImageURLs []string
}
