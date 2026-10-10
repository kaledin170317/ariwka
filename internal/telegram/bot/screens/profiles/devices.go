package profiles

import (
	"context"
	"html"
	"strconv"

	"vpn/internal/telegram/bot/screens"
)

func (s Service) Devices(ctx context.Context) (screens.View, error) {
	devices, err := s.User.Devices(ctx)
	if err != nil {
		return screens.View{}, err
	}

	rows := make([][]screens.Button, 0, len(devices)+2)
	for _, device := range devices {
		rows = append(rows, []screens.Button{{Text: device.Name, Data: "p:device:" + device.ID}})
	}

	rows = append(rows, []screens.Button{{Text: "Добавить устройство", Data: "p:device:new"}})
	rows = append(rows, []screens.Button{{Text: "← Профили", Data: "profiles"}})

	return screens.View{Text: "Отдельная подписка и ключи на каждом устройстве. Выбери устройство:", Buttons: rows}, nil
}

func (s Service) CreateDevice(ctx context.Context) (screens.View, error) {
	devices, err := s.User.Devices(ctx)
	if err != nil {
		return screens.View{}, err
	}

	device, err := s.User.CreateDevice(ctx, "Устройство "+strconv.Itoa(len(devices)+1))
	if err != nil {
		return screens.View{}, err
	}

	return deviceView(device.Name, device.URL, device.ID, device.HWID), nil
}

func (s Service) Device(ctx context.Context, id string) (screens.View, error) {
	devices, err := s.User.Devices(ctx)
	if err != nil {
		return screens.View{}, err
	}

	for _, device := range devices {
		if device.ID == id {
			return deviceView(device.Name, device.URL, device.ID, device.HWID), nil
		}
	}

	return s.Devices(ctx)
}

func (s Service) DeleteDevice(ctx context.Context, id string) (screens.View, error) {
	err := s.User.DeleteDevice(ctx, id)
	if err != nil {
		return screens.View{}, err
	}

	return s.Devices(ctx)
}

func deviceView(name, url, id, hwid string) screens.View {
	status := "Ожидает первого подключения"
	if hwid != "" {
		status = "Привязано к HWID: " + html.EscapeString(hwid)
	}

	return screens.View{
		Text: "Подписка для «" + html.EscapeString(name) + "» (все серверы с маршрутизацией):\n<code>" + html.EscapeString(url) + "</code>\n" + status,
		Buttons: [][]screens.Button{
			{{Text: "Удалить устройство", Data: "p:device:delete:" + id}},
			{{Text: "← Устройства", Data: "p:devices"}},
		},
	}
}
