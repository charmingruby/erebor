package discord

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/charmingruby/erebor/o11y/logging"
)

// Handler processes a mention or reply. An empty response sends no message.
// It may be called concurrently and must return when its context is canceled.
type Handler func(context.Context, Message) (string, error)

// Message contains application-facing message data without adding prompts.
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

func (b *Bot) handleMentionOrReply(ctx context.Context, event *discordgo.MessageCreate) {
	log := logging.LoggerFromContext(ctx)

	message, ok := parseMentionOrReply(event.Message, b.botID())
	if !ok {
		return
	}

	if !b.tryAcquireSlot() {
		log.WarnContext(ctx, "Discord bot at capacity; message ignored",
			"message_id", message.ID,
		)

		return
	}

	defer func() {
		<-b.slots
	}()

	log = log.With("message_id", message.ID)
	ctx = logging.WithLogger(ctx, log)

	handlerCtx, cancel := context.WithTimeout(ctx, b.config.HandlerTimeout)
	defer cancel()

	stopTyping := b.startTyping(handlerCtx, message.ChannelID)
	defer stopTyping()

	answer, err := b.mentionHandler(handlerCtx, message)

	cancel()
	stopTyping()

	if err != nil {
		log.ErrorContext(ctx, "handle Discord mention or reply", "error", err)
		answer = b.config.ErrorMessage
	}

	if ctx.Err() != nil || answer == "" {
		return
	}

	if err := b.replyToMessage(ctx, event.Message, answer); err != nil {
		log.ErrorContext(ctx, ErrSendReply.Error(),
			"error", err,
		)
	}
}

func parseMentionOrReply(m *discordgo.Message, botID string) (Message, bool) {
	if m.Author.Bot || m.GuildID == "" {
		return Message{}, false
	}

	mentioned := slices.ContainsFunc(m.Mentions, func(user *discordgo.User) bool {
		return user.ID == botID
	})

	replied := m.Type == discordgo.MessageTypeReply &&
		m.ReferencedMessage != nil && m.ReferencedMessage.Author != nil &&
		m.ReferencedMessage.Author.ID == botID

	if !mentioned && !replied {
		return Message{}, false
	}

	content := strings.ReplaceAll(m.Content, "<@"+botID+">", "")
	content = strings.ReplaceAll(content, "<@!"+botID+">", "")

	message := Message{
		ID:        m.ID,
		ChannelID: m.ChannelID,
		GuildID:   m.GuildID,
		AuthorID:  m.Author.ID,
		Content:   content,
	}

	if replied {
		message.ReferencedContent = m.ReferencedMessage.Content
	}

	return message, true
}

func (b *Bot) startTyping(ctx context.Context, channelID string) func() {
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})

	go func() {
		defer close(done)

		ticker := time.NewTicker(8 * time.Second)
		defer ticker.Stop()

		for ctx.Err() == nil {
			requestCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			err := b.sess.ChannelTyping(channelID, discordgo.WithContext(requestCtx))
			cancel()
			if err != nil && ctx.Err() == nil {
				logging.LoggerFromContext(ctx).DebugContext(ctx,
					"show Discord typing indicator", "channel_id", channelID, "error", err)
			}

			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()

	return func() {
		cancel()
		<-done
	}
}
