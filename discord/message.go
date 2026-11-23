package discord

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/charmingruby/erebor/discord/api"
	"github.com/charmingruby/erebor/o11y/logging"
)

func (b *Bot) handleMentionOrReply(ctx context.Context, event *discordgo.MessageCreate) {
	log := logging.LoggerFromContext(ctx)

	message, ok := parseMentionOrReply(event.Message, b.botID())
	if !ok {
		return
	}

	log = log.With("message_id", message.ID)
	ctx = logging.WithLogger(ctx, log)

	answer, err := b.executeMention(ctx, message)
	if err != nil {
		log.ErrorContext(ctx, "handle Discord mention or reply", "error", err)
		answer = b.config.errorMessage
	}

	if ctx.Err() != nil || answer == "" {
		return
	}

	if err := b.sendMessageReply(ctx, event.Message, answer); err != nil {
		log.ErrorContext(ctx, ErrSendReply.Error(),
			"error", err,
		)
	}
}

func (b *Bot) executeMention(ctx context.Context, message api.Mention) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, b.config.handlerTimeout)
	defer cancel()
	stopTyping := b.startTyping(ctx, message.ChannelID)
	defer stopTyping()
	return b.config.mentionHandler(ctx, message)
}

func parseMentionOrReply(m *discordgo.Message, botID string) (api.Mention, bool) {
	if m.Author.Bot || m.GuildID == "" {
		return api.Mention{}, false
	}

	mentioned := slices.ContainsFunc(m.Mentions, func(user *discordgo.User) bool {
		return user.ID == botID
	})

	replied := m.Type == discordgo.MessageTypeReply &&
		m.ReferencedMessage != nil && m.ReferencedMessage.Author != nil &&
		m.ReferencedMessage.Author.ID == botID

	if !mentioned && !replied {
		return api.Mention{}, false
	}

	content := strings.ReplaceAll(m.Content, "<@"+botID+">", "")
	content = strings.ReplaceAll(content, "<@!"+botID+">", "")

	message := api.Mention{
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
