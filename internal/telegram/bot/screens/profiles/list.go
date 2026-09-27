package profiles

import (
	"context"
	"html"
	"strconv"

	"vpn/internal/telegram/bot/screens"
)

func (s Service) List(ctx context.Context, index int) (screens.View, error) {
	connections, err := s.User.Connections(ctx)
	if err != nil {
		return screens.View{}, err
	}
	if len(connections) == 0 {
		return screens.View{
			Text:    "Профилей пока нет.",
			Buttons: [][]screens.Button{{{Text: "← Профили", Data: "profiles"}}},
		}, nil
	}
	if index < 0 {
		index = 0
	}
	if index >= len(connections) {
		index = len(connections) - 1
	}
	var navigation []screens.Button
	if index > 0 {
		navigation = append(navigation, screens.Button{Text: "←", Data: "p:list:" + strconv.Itoa(index-1)})
	}
	if index+1 < len(connections) {
		navigation = append(navigation, screens.Button{Text: "→", Data: "p:list:" + strconv.Itoa(index+1)})
	}
	rows := make([][]screens.Button, 0, 2)
	if len(navigation) > 0 {
		rows = append(rows, navigation)
	}
	rows = append(rows, []screens.Button{{Text: "← Профили", Data: "profiles"}})
	return screens.View{
		Text:    "Профиль " + strconv.Itoa(index+1) + " из " + strconv.Itoa(len(connections)) + ":\n<code>" + html.EscapeString(connections[index]) + "</code>",
		Buttons: rows,
	}, nil
}
