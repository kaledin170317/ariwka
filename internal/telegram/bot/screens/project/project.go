package project

import "vpn/internal/telegram/bot/screens"

const vpnDesc = `VPN сервис.
На данный момент в разработке
Если у вас возникли вопросы, проблемы или есть предложения по улучшению сервиса — пишите сюда: @kydimik`

func View() screens.View {
	return screens.View{
		Text:    vpnDesc,
		Buttons: [][]screens.Button{{{Text: "← Главная", Data: "home"}}},
	}
}
