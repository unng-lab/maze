package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Token     string
	ChannelID string
	Command   string
	WebAppURL string
	Logger    *slog.Logger
}

type Bot struct {
	token     string
	channelID string
	command   string
	webAppURL string
	logger    *slog.Logger
	client    *http.Client
	username  string
}

type apiResponse[T any] struct {
	OK          bool   `json:"ok"`
	Result      T      `json:"result"`
	Description string `json:"description"`
}

type Update struct {
	UpdateID int      `json:"update_id"`
	Message  *Message `json:"message"`
}

type Message struct {
	MessageID int    `json:"message_id"`
	Chat      Chat   `json:"chat"`
	Text      string `json:"text"`
}

type Chat struct {
	ID int64 `json:"id"`
}

type inlineKeyboardMarkup struct {
	InlineKeyboard [][]inlineKeyboardButton `json:"inline_keyboard"`
}

type inlineKeyboardButton struct {
	Text   string     `json:"text"`
	WebApp *webAppRef `json:"web_app,omitempty"`
}

type webAppRef struct {
	URL string `json:"url"`
}

type sendMessagePayload struct {
	ChatID      any                   `json:"chat_id"`
	Text        string                `json:"text"`
	ReplyMarkup *inlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

func New(cfg Config) (*Bot, error) {
	if cfg.Token == "" {
		return nil, errors.New("telegram token is required")
	}

	if cfg.ChannelID == "" {
		return nil, errors.New("channel identifier is required")
	}

	if cfg.WebAppURL == "" {
		return nil, errors.New("web app URL is required")
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	command := cfg.Command
	if command == "" {
		command = "maze"
	}

	bot := &Bot{
		token:     cfg.Token,
		channelID: cfg.ChannelID,
		command:   command,
		webAppURL: cfg.WebAppURL,
		logger:    logger,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	me, err := bot.getMe(ctx)
	if err != nil {
		return nil, fmt.Errorf("get bot info: %w", err)
	}

	bot.username = me.Username
	bot.logger.Info("telegram bot initialised", "username", bot.username)

	return bot, nil
}

type user struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

func (b *Bot) Run(ctx context.Context) error {
	offset := 0

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		updates, err := b.getUpdates(ctx, offset)
		if err != nil {
			b.logger.Error("get updates", "error", err)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(time.Second):
			}
			continue
		}

		for _, update := range updates {
			offset = update.UpdateID + 1
			b.handleUpdate(ctx, update)
		}
	}
}

func (b *Bot) handleUpdate(ctx context.Context, update Update) {
	if update.Message == nil {
		return
	}

	text := strings.TrimSpace(update.Message.Text)
	if text == "" || !strings.HasPrefix(text, "/") {
		return
	}

	command := b.extractCommand(text)
	if command != b.command {
		return
	}

	if err := b.publishMaze(ctx, update.Message.Chat.ID); err != nil {
		b.logger.Error("publish maze", "error", err)
		_ = b.sendMessage(ctx, update.Message.Chat.ID, "Не удалось опубликовать лабиринт. Попробуйте позже.", nil)
		return
	}

	if err := b.sendMessage(ctx, update.Message.Chat.ID, "Лабиринт опубликован в канале.", nil); err != nil {
		b.logger.Error("send confirmation", "error", err)
	}
}

func (b *Bot) extractCommand(text string) string {
	first := strings.Fields(text)
	if len(first) == 0 {
		return ""
	}

	command := strings.TrimPrefix(first[0], "/")

	if at := strings.IndexRune(command, '@'); at != -1 {
		name := command[at+1:]
		if !strings.EqualFold(name, b.username) {
			return ""
		}
		command = command[:at]
	}

	return command
}

func (b *Bot) publishMaze(ctx context.Context, requester int64) error {
	keyboard := &inlineKeyboardMarkup{
		InlineKeyboard: [][]inlineKeyboardButton{
			{
				{
					Text:   "Открыть лабиринт",
					WebApp: &webAppRef{URL: b.webAppURL},
				},
			},
		},
	}

	if err := b.sendMessage(ctx, b.channelID, "🧩 Новый лабиринт доступен в мини-приложении. Нажмите кнопку, чтобы открыть его.", keyboard); err != nil {
		return err
	}

	b.logger.Info("maze published", "channel", b.channelID, "requested_by", requester)
	return nil
}

func (b *Bot) sendMessage(ctx context.Context, chatID any, text string, markup *inlineKeyboardMarkup) error {
	payload := sendMessagePayload{
		ChatID:      chatID,
		Text:        text,
		ReplyMarkup: markup,
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var resp apiResponse[Message]
	if err := b.post(ctx, "sendMessage", payload, &resp); err != nil {
		return err
	}

	if !resp.OK {
		return fmt.Errorf("telegram sendMessage failed: %s", resp.Description)
	}

	return nil
}

func (b *Bot) getUpdates(ctx context.Context, offset int) ([]Update, error) {
	params := url.Values{}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	params.Set("timeout", "30")
	params.Set("allowed_updates", "[\"message\"]")

	ctx, cancel := context.WithTimeout(ctx, 35*time.Second)
	defer cancel()

	var resp apiResponse[[]Update]
	if err := b.get(ctx, "getUpdates", params, &resp); err != nil {
		return nil, err
	}

	if !resp.OK {
		return nil, fmt.Errorf("telegram getUpdates failed: %s", resp.Description)
	}

	return resp.Result, nil
}

func (b *Bot) getMe(ctx context.Context) (user, error) {
	var resp apiResponse[user]
	if err := b.get(ctx, "getMe", nil, &resp); err != nil {
		return user{}, err
	}

	if !resp.OK {
		return user{}, fmt.Errorf("telegram getMe failed: %s", resp.Description)
	}

	return resp.Result, nil
}

func (b *Bot) apiURL(method string) string {
	return fmt.Sprintf("https://api.telegram.org/bot%s/%s", b.token, method)
}

func (b *Bot) get(ctx context.Context, method string, params url.Values, result any) error {
	endpoint := b.apiURL(method)
	if params != nil {
		endpoint = endpoint + "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("telegram returned status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	return nil
}

func (b *Bot) post(ctx context.Context, method string, payload any, result any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.apiURL(method), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("telegram returned status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	return nil
}
