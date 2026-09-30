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
		Text: "Ссылка подписки Happ отправлена отдельным сообщением. Добавь её в Happ как подписку.",
		Buttons: [][]screens.Button{
			{{Text: "← Профили", Data: "profiles"}},
		},
		Notice: "<code>" + html.EscapeString(url) + "</code>",
	}, nil
}
