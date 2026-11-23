package discord

import (
	"time"

	"github.com/charmingruby/erebor/discord/api"
)

type config struct {
	mentionHandler api.MentionHandler
	errorMessage   string
	commandGuildID string
	commands       []api.Command
	httpTimeout    time.Duration
	handlerTimeout time.Duration
	replyTimeout   time.Duration
}

// Option configures a bot before its session is created.
type Option func(*config)

// WithCommands adds slash commands. Repeated calls append to the command list.
// Definitions must not be mutated after New.
func WithCommands(commands ...api.Command) Option {
	return func(c *config) {
		c.commands = append(c.commands, commands...)
	}
}

// WithMentionHandler enables guild mentions and replies to the bot's messages.
// A nil handler disables both.
func WithMentionHandler(handler api.MentionHandler) Option {
	return func(c *config) {
		c.mentionHandler = handler
	}
}

// WithErrorMessage sets the fallback used when a handler fails. Empty disables it.
func WithErrorMessage(message string) Option {
	return func(c *config) {
		c.errorMessage = message
	}
}

// WithCommandGuild selects a development guild. Listen clears all global commands
// and replaces this guild's commands. Use an application separate from production.
// Empty registers globally without deleting existing commands.
func WithCommandGuild(guildID string) Option {
	return func(c *config) {
		c.commandGuildID = guildID
	}
}

// WithHTTPTimeout sets a positive request timeout. Default: 15 seconds.
func WithHTTPTimeout(timeout time.Duration) Option {
	return func(c *config) {
		c.httpTimeout = timeout
	}
}

// WithHandlerTimeout sets a positive handler timeout. Default: 3 minutes.
// Commands cap execution at 14 minutes because of Discord's token lifetime.
func WithHandlerTimeout(timeout time.Duration) Option {
	return func(c *config) {
		c.handlerTimeout = timeout
	}
}

// WithReplyTimeout sets a positive timeout for all reply chunks. Default: 1 minute.
func WithReplyTimeout(timeout time.Duration) Option {
	return func(c *config) {
		c.replyTimeout = timeout
	}
}
