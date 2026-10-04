package profiles

import (
	"context"
	"html"
	"strconv"

	"vpn/internal/telegram/bot/screens"
)

func (s Service) List(ctx context.Context, index int) (screens.View, error) {
	profiles, err := s.User.Profiles(ctx)
	if err != nil {
		return screens.View{}, err
	}

	if len(profiles) == 0 {
		return screens.View{
			Text:    "Профилей пока нет.",
			Buttons: [][]screens.Button{{{Text: "← Профили", Data: "profiles"}}},
		}, nil
	}

	if index < 0 {
		index = 0
	}

	if index >= len(profiles) {
		index = len(profiles) - 1
	}

	var navigation []screens.Button
	if index > 0 {
		navigation = append(navigation, screens.Button{Text: "←", Data: "p:list:" + strconv.Itoa(index-1)})
	}

	if index+1 < len(profiles) {
		navigation = append(navigation, screens.Button{Text: "→", Data: "p:list:" + strconv.Itoa(index+1)})
	}

	rows := make([][]screens.Button, 0, 3)
	if len(navigation) > 0 {
		rows = append(rows, navigation)
	}

	rows = append(rows, []screens.Button{{Text: "Удалить профиль", Data: "p:delete:" + profiles[index].ID}})
	rows = append(rows, []screens.Button{{Text: "← Профили", Data: "profiles"}})

	return screens.View{
		Text:    "Профиль " + strconv.Itoa(index+1) + " из " + strconv.Itoa(len(profiles)) + ":\n<code>" + html.EscapeString(profiles[index].Connection) + "</code>",
		Buttons: rows,
	}, nil
}

func (s Service) Delete(ctx context.Context, id string) (screens.View, error) {
	err := s.User.DeleteProfile(ctx, id)
	if err != nil {
		return screens.View{}, err
	}

	view, err := s.List(ctx, 0)
	if err != nil {
		return screens.View{}, err
	}

	view.Text = "Профиль удалён.\n\n" + view.Text

	return view, nil
}
