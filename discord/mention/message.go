package mention

import "context"

// Handler processes a mention or reply. Empty output sends no message.
// It may run concurrently and must honor context cancellation.
type Handler func(context.Context, Message) (string, error)

// Message contains the application-facing data of a mention or reply.
type Message struct {
	ID        string
	ChannelID string
	GuildID   string
	AuthorID  string
	// Content excludes mentions of this bot; other whitespace is preserved.
	Content string
	// ReferencedContent contains the bot response being replied to, if any.
	ReferencedContent string
}
