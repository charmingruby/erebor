package discord

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/charmingruby/erebor/discord/command"
	"github.com/charmingruby/erebor/discord/mention"
	"github.com/charmingruby/erebor/o11y/logging"
)

var (
	// ErrInvalidConfig indicates a missing token, nonpositive timeout, or empty bot.
	ErrInvalidConfig = errors.New("invalid Discord bot configuration")
	// ErrCreateSession indicates that DiscordGo could not create a session.
	ErrCreateSession = errors.New("create Discord bot session")
	// ErrConnect indicates a gateway connection failure.
	ErrConnect = errors.New("connect Discord bot")
	// ErrCloseSession indicates a session shutdown failure.
	ErrCloseSession = errors.New("close Discord bot session")
	// ErrSendReply indicates a failed or canceled reply.
	ErrSendReply = errors.New("send Discord bot reply")
)

type config struct {
	ErrorMessage   string
	CommandGuildID string
	HTTPTimeout    time.Duration
	HandlerTimeout time.Duration
	ReplyTimeout   time.Duration
}

// Bot handles slash commands and optional guild mentions. Construct it with NewBot.
// Configure it before listening, and call Listen only once per bot.
// Configuration and registration are not safe for concurrent use.
type Bot struct {
	sess           *discordgo.Session
	mentionHandler mention.Handler
	commands       map[string]command.Definition
	config         config
}

// NewBot creates a bot without opening a connection. Token is the raw bot token.
// Mentions are disabled unless WithMentionReplyHandler is supplied.
// Register commands before calling Listen.
func NewBot(token string, opts ...BotOption) (*Bot, error) {
	b := &Bot{
		config: config{
			HTTPTimeout:    15 * time.Second,
			HandlerTimeout: 3 * time.Minute,
			ReplyTimeout:   time.Minute,
		},
		commands: make(map[string]command.Definition),
	}

	for _, opt := range opts {
		opt(b)
	}

	if token == "" ||
		b.config.HTTPTimeout <= 0 ||
		b.config.HandlerTimeout <= 0 ||
		b.config.ReplyTimeout <= 0 {
		return nil, ErrInvalidConfig
	}

	sess, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCreateSession, err)
	}

	sess.Identify.Intents = 0
	if b.mentionHandler != nil {
		sess.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent
	}

	sess.Client.Timeout = b.config.HTTPTimeout

	b.sess = sess

	return b, nil
}

func (b *Bot) botID() string {
	b.sess.State.RLock()
	defer b.sess.State.RUnlock()

	if b.sess.State.User == nil {
		return ""
	}

	return b.sess.State.User.ID
}

// Listen connects and registers commands, then handles interactions and mentions
// until ctx is canceled. It cancels active handlers and waits for them to finish.
// Normal cancellation returns nil. Call Listen only once per bot, after all
// configuration and command registration are complete.
func (b *Bot) Listen(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if b.mentionHandler == nil && len(b.commands) == 0 {
		return ErrInvalidConfig
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var callbackMu sync.Mutex
	var callbacks sync.WaitGroup

	// Serialize callback admission with shutdown so Add cannot race with Wait
	// after cancellation, including callbacks already dispatched by the session.
	runTrackedHandler := func(handle func()) {
		callbackMu.Lock()

		if ctx.Err() != nil {
			callbackMu.Unlock()
			return
		}

		callbacks.Add(1)

		callbackMu.Unlock()

		defer callbacks.Done()
		handle()
	}

	var removers []func()

	if b.mentionHandler != nil {
		removers = append(removers, b.sess.AddHandler(func(_ *discordgo.Session, event *discordgo.MessageCreate) {
			runTrackedHandler(func() {
				b.handleMentionOrReply(ctx, event)
			})
		}))
	}

	if len(b.commands) != 0 {
		removers = append(removers, b.sess.AddHandler(func(_ *discordgo.Session, event *discordgo.InteractionCreate) {
			runTrackedHandler(func() {
				b.handleCommand(ctx, event)
			})
		}))
	}

	err := b.runSession(ctx)

	for _, remove := range removers {
		remove()
	}

	callbackMu.Lock()
	cancel()
	callbackMu.Unlock()

	closeErr := b.sess.Close()
	callbacks.Wait()
	if closeErr != nil {
		return errors.Join(err, fmt.Errorf("%w: %w", ErrCloseSession, closeErr))
	}

	return err
}

func (b *Bot) runSession(ctx context.Context) error {
	if err := b.sess.Open(); err != nil {
		return fmt.Errorf("%w: %w", ErrConnect, err)
	}

	if err := b.publishCommands(ctx); err != nil {
		return err
	}

	logging.LoggerFromContext(ctx).InfoContext(ctx, "Discord bot connected")
	<-ctx.Done()

	return nil
}
