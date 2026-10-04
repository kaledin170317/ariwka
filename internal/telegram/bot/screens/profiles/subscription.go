package profiles

import (
	"context"
	"html"

	"vpn/internal/telegram/bot/screens"
)

func (s Service) Subscription(ctx context.Context) (screens.View, error) {
	url, err := s.User.SubscriptionLink(ctx)
	if err != nil {
		return screens.View{}, err
	}

	return screens.View{
		Text: "Добавь эту ссылку в Happ как подписку:\n<code>" + html.EscapeString(url) + "</code>",
		Buttons: [][]screens.Button{
			{{Text: "← Профили", Data: "profiles"}},
		},
	}, nil
}
