package profiles

import (
	"vpn/internal/telegram/bot/provider"
	"vpn/internal/telegram/bot/screens"
)

type Service struct {
	User provider.User
}

func Menu() screens.View {
	return screens.View{
		Text: "Конфиги. Выбери действие:",
		Buttons: [][]screens.Button{
			{{Text: "Мои конфиги", Data: "p:devices"}},
			{{Text: "Создать конфиг", Data: "p:create"}},
			{{Text: "← Главная", Data: "home"}},
		},
	}
}
