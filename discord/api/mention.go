package api

import "context"

// MentionHandler processes a mention or reply. Empty output sends no message.
// It may run concurrently and must honor context cancellation.
type MentionHandler func(context.Context, Mention) (string, error)

// Mention contains the application-facing data of a mention or reply.
type Mention struct {
	ID        string
	ChannelID string
	GuildID   string
	AuthorID  string
	// Content excludes mentions of this bot; other whitespace is preserved.
	Content string
	// ReferencedContent contains the bot response being replied to, if any.
	ReferencedContent string
}
