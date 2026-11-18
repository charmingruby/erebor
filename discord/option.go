package discord

import (
	"time"

	"github.com/charmingruby/erebor/discord/mention"
)

// BotOption configures a bot before its session is created.
type BotOption func(*Bot)

// WithMentionReplyHandler handles guild mentions and replies to the bot's
// messages. A nil handler disables both; slash commands are unaffected.
func WithMentionReplyHandler(handler mention.Handler) BotOption {
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

// WithCommandGuild limits command registration to a guild. On Listen, a nonempty
// ID removes all global commands and replaces all commands in that guild.
// This affects every instance sharing the Discord application, including production.
// Commands in other guilds are not removed. Empty means global registration
// without deleting existing commands.
func WithCommandGuild(guildID string) BotOption {
	return func(b *Bot) {
		b.config.CommandGuildID = guildID
	}
}

// WithHTTPTimeout sets a positive HTTP timeout. The default is 15 seconds.
func WithHTTPTimeout(timeout time.Duration) BotOption {
	return func(b *Bot) {
		b.config.HTTPTimeout = timeout
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
