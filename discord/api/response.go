package api

// Response contains command reply text and an optional image displayed below it.
// ImageURL must be a publicly accessible HTTP(S) image URL.
// An empty response removes the deferred command response.
type Response struct {
	Content  string
	ImageURL string
}
