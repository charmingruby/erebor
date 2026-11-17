package discord

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/charmingruby/erebor/o11y/logging"
)

var (
	// ErrInvalidConfig indicates a missing token, nonpositive limit, or empty bot.
	ErrInvalidConfig = errors.New("invalid Discord bot configuration")
	// ErrCreateSession indicates that DiscordGo could not create a session.
	ErrCreateSession = errors.New("create Discord bot session")
	// ErrConnect indicates a gateway connection failure.
	ErrConnect = errors.New("connect Discord bot")
	// ErrCloseSession indicates a session shutdown failure.
	ErrCloseSession = errors.New("close Discord bot session")
	// ErrAlreadyRunning indicates a concurrent listener on the same bot.
	ErrAlreadyRunning = errors.New("discord bot already running")
	// ErrSendReply indicates a failed or canceled reply.
	ErrSendReply = errors.New("send Discord bot reply")
)

type config struct {
	ErrorMessage          string
	CommandGuildID        string
	MaxConcurrentRequests int
	RequestTimeout        time.Duration
	HandlerTimeout        time.Duration
	ReplyTimeout          time.Duration
}

// Bot handles slash commands and optional guild mentions. Construct it with NewBot.
type Bot struct {
	sess           *discordgo.Session
	mentionHandler Handler
	slots          chan struct{}
	commands       map[string]Command
	config         config
	mu             sync.Mutex
	running        bool
}

// NewBot creates a bot without opening a connection. Token is the raw bot token.
// Mentions are disabled unless WithHandler is supplied.
// Register commands before calling Listen.
func NewBot(token string, opts ...BotOption) (*Bot, error) {
	b := &Bot{
		config: config{
			MaxConcurrentRequests: 8,
			RequestTimeout:        15 * time.Second,
			HandlerTimeout:        3 * time.Minute,
			ReplyTimeout:          time.Minute,
		},
		commands: make(map[string]Command),
	}

	for _, opt := range opts {
		opt(b)
	}

	if token == "" ||
		b.config.MaxConcurrentRequests <= 0 ||
		b.config.RequestTimeout <= 0 ||
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

	sess.Client.Timeout = b.config.RequestTimeout

	b.sess = sess
	b.slots = make(chan struct{}, b.config.MaxConcurrentRequests)

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
// Normal cancellation returns nil. Concurrent calls return ErrAlreadyRunning.
func (b *Bot) Listen(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	b.mu.Lock()
	if b.running {
		b.mu.Unlock()
		return ErrAlreadyRunning
	}

	if b.mentionHandler == nil && len(b.commands) == 0 {
		b.mu.Unlock()
		return ErrInvalidConfig
	}

	b.running = true
	b.mu.Unlock()

	defer func() {
		b.mu.Lock()
		b.running = false
		b.mu.Unlock()
	}()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var callbackMu sync.Mutex
	var callbacks sync.WaitGroup

	trackCallback := func(handle func()) {
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
			trackCallback(func() {
				b.handleMentionOrReply(ctx, event)
			})
		}))
	}

	if len(b.commands) != 0 {
		removers = append(removers, b.sess.AddHandler(func(_ *discordgo.Session, event *discordgo.InteractionCreate) {
			trackCallback(func() {
				b.handleCommand(ctx, event)
			})
		}))
	}

	err := b.connectAndWait(ctx)

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

func (b *Bot) connectAndWait(ctx context.Context) error {
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

func (b *Bot) tryAcquireSlot() bool {
	select {
	case b.slots <- struct{}{}:
		return true
	default:
		return false
	}
}
