package discord

import "time"

// BotOption configures a bot before its session is created.
type BotOption func(*Bot)

// WithHandler enables mentions and replies. A nil handler disables them.
func WithHandler(handler Handler) BotOption {
	return func(b *Bot) {
		b.mentionHandler = handler
	}
}

// WithErrorMessage sets the reply used when a handler fails. Empty disables it.
func WithErrorMessage(message string) BotOption {
	return func(b *Bot) {
		b.config.ErrorMessage = message
	}
}

// WithCommandGuildID limits command registration to a guild. Empty means global.
func WithCommandGuildID(guildID string) BotOption {
	return func(b *Bot) {
		b.config.CommandGuildID = guildID
	}
}

// WithMaxConcurrentRequests sets a positive handler limit. The default is 8.
func WithMaxConcurrentRequests(limit int) BotOption {
	return func(b *Bot) {
		b.config.MaxConcurrentRequests = limit
	}
}

// WithRequestTimeout sets a positive HTTP timeout. The default is 15 seconds.
func WithRequestTimeout(timeout time.Duration) BotOption {
	return func(b *Bot) {
		b.config.RequestTimeout = timeout
	}
}

// WithHandlerTimeout sets a positive handler timeout. The default is 3 minutes.
// Commands cap this timeout at 14 minutes.
func WithHandlerTimeout(timeout time.Duration) BotOption {
	return func(b *Bot) {
		b.config.HandlerTimeout = timeout
	}
}

// WithReplyTimeout sets a positive timeout for all reply chunks.
// The default is 1 minute.
func WithReplyTimeout(timeout time.Duration) BotOption {
	return func(b *Bot) {
		b.config.ReplyTimeout = timeout
	}
}
