package discord

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/charmingruby/erebor/discord/api"
)

type replyTransport func(*http.Request) (*http.Response, error)

func (transport replyTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func TestSendCommandReply(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		imageCount int
		wantText   []string
		wantImages []int
	}{
		{name: "empty response"},
		{name: "text only", content: "hello", wantText: []string{"hello"}, wantImages: []int{0}},
		{name: "image only", imageCount: 1, wantText: []string{""}, wantImages: []int{1}},
		{name: "ten images", content: "hello", imageCount: 10, wantText: []string{"hello"}, wantImages: []int{10}},
		{name: "multiple image batches", imageCount: 21, wantText: []string{"", "", ""}, wantImages: []int{10, 10, 1}},
		{
			name: "more text than images", content: strings.Repeat("a", 2001), imageCount: 1,
			wantText: []string{strings.Repeat("a", 2000), "a"}, wantImages: []int{1, 0},
		},
		{
			name: "more images than text", content: "hello", imageCount: 11,
			wantText: []string{"hello", ""}, wantImages: []int{10, 1},
		},
		{
			name: "unicode text and images", content: strings.Repeat("😀", 1001), imageCount: 11,
			wantText: []string{strings.Repeat("😀", 1000), "😀"}, wantImages: []int{10, 1},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var urls []string
			for i := range test.imageCount {
				urls = append(urls, fmt.Sprintf("https://example.com/%d.png", i))
			}
			var gotText, gotURLs []string
			var gotImages []int
			calls := 0
			session := &discordgo.Session{Ratelimiter: discordgo.NewRatelimiter(), Client: &http.Client{
				Transport: replyTransport(func(request *http.Request) (*http.Response, error) {
					if request.Body != nil {
						defer request.Body.Close()
					}
					index := calls
					calls++
					wantMethod := http.MethodPost
					if index == 0 {
						wantMethod = http.MethodPatch
						if len(test.wantText) == 0 {
							wantMethod = http.MethodDelete
						}
					}
					if request.Method != wantMethod {
						t.Errorf("method = %s, want %s", request.Method, wantMethod)
					}
					if request.Method == http.MethodDelete {
						return &http.Response{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
					}
					var payload struct {
						Content         string                            `json:"content"`
						Embeds          []*discordgo.MessageEmbed         `json:"embeds"`
						Flags           discordgo.MessageFlags            `json:"flags"`
						AllowedMentions *discordgo.MessageAllowedMentions `json:"allowed_mentions"`
					}
					if err := json.UnmarshalRead(request.Body, &payload); err != nil {
						t.Fatal(err)
					}
					if index > 0 && payload.Flags != discordgo.MessageFlagsEphemeral {
						t.Errorf("follow-up flags = %d, want ephemeral", payload.Flags)
					}
					if payload.AllowedMentions == nil || len(payload.AllowedMentions.Parse) != 0 {
						t.Error("expected mentions to be disabled")
					}
					gotText = append(gotText, payload.Content)
					gotImages = append(gotImages, len(payload.Embeds))
					for _, embed := range payload.Embeds {
						gotURLs = append(gotURLs, embed.Image.URL)
					}
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(`{"id":"reply"}`)),
						Header:     make(http.Header),
					}, nil
				}),
			}}
			bot := &Bot{sess: session, config: config{replyTimeout: time.Second}}
			err := bot.sendCommandReply(t.Context(), &discordgo.Interaction{AppID: "app", Token: "token"},
				api.Response{Content: test.content, ImageURLs: urls}, discordgo.MessageFlagsEphemeral)
			if err != nil {
				t.Fatal(err)
			}
			if calls != max(1, len(test.wantText)) {
				t.Errorf("calls = %d, want %d", calls, max(1, len(test.wantText)))
			}
			if !slices.Equal(gotText, test.wantText) || !slices.Equal(gotImages, test.wantImages) {
				t.Errorf("messages = %q / %v images, want %q / %v images", gotText, gotImages, test.wantText, test.wantImages)
			}
			if !slices.Equal(gotURLs, urls) {
				t.Errorf("image URLs = %v, want %v", gotURLs, urls)
			}
		})
	}
}

func TestSendCommandReplyError(t *testing.T) {
	cause := errors.New("request failed")
	bot := &Bot{
		sess: &discordgo.Session{Ratelimiter: discordgo.NewRatelimiter(), Client: &http.Client{
			Transport: replyTransport(func(*http.Request) (*http.Response, error) {
				return nil, cause
			}),
		}},
		config: config{replyTimeout: time.Second},
	}
	err := bot.sendCommandReply(t.Context(), &discordgo.Interaction{AppID: "app", Token: "token"},
		api.Response{ImageURLs: []string{"https://example.com/1.png"}}, 0)
	if !errors.Is(err, ErrSendReply) || !errors.Is(err, cause) {
		t.Fatalf("error = %v, want ErrSendReply and transport error", err)
	}
}
