package bot

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strconv"

	"vpn/internal/telegram/bot/screens"
)

func (b telegram) render(ctx context.Context, chat, id int64, view screens.View) error {
	markup, err := json.Marshal(struct {
		InlineKeyboard [][]screens.Button `json:"inline_keyboard"`
	}{view.Buttons})
	if err != nil {
		return err
	}
	values := url.Values{
		"chat_id":      {strconv.FormatInt(chat, 10)},
		"text":         {view.Text},
		"parse_mode":   {"HTML"},
		"reply_markup": {string(markup)},
	}
	if id != 0 {
		values.Set("message_id", strconv.FormatInt(id, 10))
		var result json.RawMessage
		err = b.call(ctx, "editMessageText", values, &result)
		if err == nil {
			b.messages[chat] = id
			return nil
		}
		if !errors.Is(err, errMessageMissing) {
			return err
		}
		values.Del("message_id")
	}
	var message struct {
		ID int64 `json:"message_id"`
	}
	if err := b.call(ctx, "sendMessage", values, &message); err != nil {
		return err
	}
	b.messages[chat] = message.ID
	return nil
}
