package bot

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"time"

	"vpn/internal/telegram/bot/screens"
	"vpn/internal/telegram/bot/screens/profiles"
	"vpn/internal/telegram/bot/screens/project"
)

func (b telegram) handle(ctx context.Context, u update) error {
	if c := u.Callback; c != nil {
		if c.Message == nil || c.Message.Chat.Type != "private" || c.From.ID != c.Message.Chat.ID {
			return nil
		}
		chat, id := c.Message.Chat.ID, c.Message.ID
		answer := url.Values{"callback_query_id": {c.ID}}
		if current := b.messages[chat]; current != 0 && current != id {
			answer.Set("text", "Открой актуальное сообщение бота")
			var result json.RawMessage
			return b.call(ctx, "answerCallbackQuery", answer, &result)
		}
		var result json.RawMessage
		_ = b.call(ctx, "answerCallbackQuery", answer, &result)
		return b.action(ctx, c.From.ID, chat, id, c.Data)
	}
	msg := u.Message
	if msg == nil || msg.Chat.Type != "private" || msg.From.ID != msg.Chat.ID {
		return nil
	}
	if b.messages[msg.Chat.ID] == 0 {
		if err := b.removeKeyboard(ctx, msg.Chat.ID); err != nil {
			return err
		}
	}
	var result json.RawMessage
	_ = b.call(ctx, "deleteMessage", url.Values{
		"chat_id":    {strconv.FormatInt(msg.Chat.ID, 10)},
		"message_id": {strconv.FormatInt(msg.ID, 10)},
	}, &result)
	data := "home"
	if fields := strings.Fields(msg.Text); len(fields) > 0 {
		switch strings.Split(fields[0], "@")[0] {
		case "/project":
			data = "project"
		case "/profiles":
			data = "profiles"
		case "/configs":
			data = "p:list:0"
		case "/create":
			data = "p:create"
		}
	}
	switch msg.Text {
	case "Информация о проекте":
		data = "project"
	case "Профили":
		data = "profiles"
	case "Список профилей":
		data = "p:list:0"
	case "Создать профиль":
		data = "p:create"
	}
	return b.action(ctx, msg.From.ID, msg.Chat.ID, b.messages[msg.Chat.ID], data)
}

func (b telegram) removeKeyboard(ctx context.Context, chat int64) error {
	var message struct {
		ID int64 `json:"message_id"`
	}
	if err := b.call(ctx, "sendMessage", url.Values{
		"chat_id":      {strconv.FormatInt(chat, 10)},
		"text":         {"Открываю меню…"},
		"reply_markup": {`{"remove_keyboard":true}`},
	}, &message); err != nil {
		return err
	}
	var result json.RawMessage
	_ = b.call(ctx, "deleteMessage", url.Values{
		"chat_id":    {strconv.FormatInt(chat, 10)},
		"message_id": {strconv.FormatInt(message.ID, 10)},
	}, &result)
	return nil
}

func (b telegram) action(ctx context.Context, userID, chat, id int64, data string) error {
	user, err := b.control.ForUser(ctx, userID)
	if err != nil {
		return err
	}
	requestCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	p := profiles.Service{User: user}
	var view screens.View
	switch {
	case data == "home":
		view = screens.Home()
	case data == "project":
		view = project.View()
	case data == "profiles":
		view = profiles.Menu()
	case data == "p:create":
		view, err = p.NodesList(requestCtx)
	case strings.HasPrefix(data, "p:list:"):
		page, parseErr := strconv.Atoi(strings.TrimPrefix(data, "p:list:"))
		if parseErr != nil {
			view = profiles.Menu()
		} else {
			view, err = p.List(requestCtx, page)
		}
	case strings.HasPrefix(data, "p:node:"):
		view, err = p.Inbounds(requestCtx, strings.TrimPrefix(data, "p:node:"))
	case strings.HasPrefix(data, "p:new:"):
		parts := strings.SplitN(strings.TrimPrefix(data, "p:new:"), ":", 2)
		if len(parts) != 2 {
			view = profiles.Menu()
		} else {
			view, err = p.Create(requestCtx, parts[0], parts[1])
		}
	default:
		view = screens.Home()
	}
	if err != nil {
		return err
	}
	if err := b.render(ctx, chat, id, view); err != nil {
		return err
	}
	if view.Notice != "" {
		return b.send(ctx, chat, view.Notice)
	}
	return nil
}
