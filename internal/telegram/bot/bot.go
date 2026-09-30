package bot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"vpn/internal/telegram/bot/provider"
)

type update struct {
	ID       int64    `json:"update_id"`
	Message  *message `json:"message"`
	Callback *struct {
		ID   string `json:"id"`
		Data string `json:"data"`
		From struct {
			ID int64 `json:"id"`
		} `json:"from"`
		Message *message `json:"message"`
	} `json:"callback_query"`
}

type message struct {
	ID   int64  `json:"message_id"`
	Text string `json:"text"`
	Chat struct {
		ID   int64  `json:"id"`
		Type string `json:"type"`
	} `json:"chat"`
	From struct {
		ID int64 `json:"id"`
	} `json:"from"`
}

type telegram struct {
	endpoint string
	http     *http.Client
	control  provider.Control
	messages map[int64]int64
}

var errMessageMissing = errors.New("main message is missing")

func (b telegram) call(ctx context.Context, method string, values url.Values, result any) error {
	ctx, cancel := context.WithTimeout(ctx, 40*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.endpoint+method, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := b.http.Do(req)
	if err != nil {
		return errors.New("telegram connection failed")
	}
	defer resp.Body.Close()

	var body struct {
		OK          bool            `json:"ok"`
		Result      json.RawMessage `json:"result"`
		Description string          `json:"description"`
	}

	err = json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		return err
	}

	if !body.OK && method == "editMessageText" {
		if strings.Contains(body.Description, "message is not modified") {
			return nil
		}

		if strings.Contains(body.Description, "message to edit not found") || strings.Contains(body.Description, "message not found") || strings.Contains(body.Description, "message can't be edited") {
			return errMessageMissing
		}
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram HTTP %d", resp.StatusCode)
	}

	if !body.OK {
		return errors.New("telegram API error")
	}

	return json.Unmarshal(body.Result, result)
}

func (b telegram) send(ctx context.Context, chat int64, text string) error {
	var result json.RawMessage

	return b.call(ctx, "sendMessage", url.Values{
		"chat_id":    {strconv.FormatInt(chat, 10)},
		"text":       {text},
		"parse_mode": {"HTML"},
	}, &result)
}

func Run(ctx context.Context, token, username, password, address string) error {
	if token == "" || username == "" || password == "" || address == "" {
		return errors.New("TELEGRAM_BOT_TOKEN, TELEGRAM_CONTROL_USERNAME, TELEGRAM_CONTROL_PASSWORD and CONTROL_ADDRESS required")
	}

	control, err := provider.Open(address, username, password)
	if err != nil {
		return err
	}
	defer control.Close()

	b := telegram{endpoint: "https://api.telegram.org/bot" + token + "/", http: &http.Client{},
		control: control, messages: make(map[int64]int64)}

	var updates []update

	err = b.call(ctx, "getUpdates", url.Values{"offset": {"-1"}, "timeout": {"0"}, "allowed_updates": {`["message","callback_query"]`}}, &updates)
	if err != nil {
		return err
	}

	var offset int64

	for ctx.Err() == nil {
		if len(updates) == 0 {
			err := b.call(ctx, "getUpdates", url.Values{
				"offset": {strconv.FormatInt(offset, 10)}, "timeout": {"25"},
				"allowed_updates": {`["message","callback_query"]`},
			}, &updates)
			if err != nil {
				log.Print(err)
				time.Sleep(3 * time.Second)

				continue
			}
		}

		for _, u := range updates {
			err := b.handle(ctx, u)
			if err != nil {
				log.Print(err)

				chat := int64(0)
				if u.Message != nil {
					chat = u.Message.Chat.ID
				}

				if u.Callback != nil && u.Callback.Message != nil {
					chat = u.Callback.Message.Chat.ID
				}

				if chat != 0 {
					sendErr := b.send(ctx, chat, "Не удалось выполнить команду. Попробуй позже.")
					if sendErr != nil {
						log.Print(sendErr)
					}
				}
			}

			offset = u.ID + 1
		}

		updates = nil
	}

	return ctx.Err()
}
