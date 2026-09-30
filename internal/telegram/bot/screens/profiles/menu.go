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
		Text: "Профили. Выбери действие:",
		Buttons: [][]screens.Button{
			{{Text: "Список профилей", Data: "p:list:0"}},
			{{Text: "Создать профиль", Data: "p:create"}},
			{{Text: "Подписка Happ", Data: "p:subscription"}},
			{{Text: "← Главная", Data: "home"}},
		},
	}
}
