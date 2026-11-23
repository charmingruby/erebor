package api

// Response contains command reply text and an optional image gallery below it.
// ImageURLs must be publicly accessible HTTP(S) image URLs.
// Images are sent in order, with at most 10 per gallery/message.
// Replies use Discord Components V2 with TextDisplay and MediaGallery components.
// An empty response removes the deferred command response.
type Response struct {
	Content   string
	ImageURLs []string
}
