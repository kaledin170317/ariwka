package profiles

import (
	"context"
	"html"
	"strconv"
	"strings"

	"vpn/internal/telegram/bot/screens"
)

func (s Service) Devices(ctx context.Context) (screens.View, error) {
	devices, err := s.User.Devices(ctx)
	if err != nil {
		return screens.View{}, err
	}

	rows := make([][]screens.Button, 0, len(devices)+2)
	for _, device := range devices {
		mode := "с маршрутизацией"
		if device.DefaultNetwork {
			mode = "без маршрутизации"
		}

		rows = append(rows, []screens.Button{{Text: device.Name + " · " + mode, Data: "p:device:" + device.ID}})
	}

	rows = append(rows, []screens.Button{{Text: "Создать конфиг", Data: "p:create"}})
	rows = append(rows, []screens.Button{{Text: "← Профили", Data: "profiles"}})

	return screens.View{Text: "Конфиги устройств (каждый содержит все серверы):", Buttons: rows}, nil
}

func CreateMode() screens.View {
	return screens.View{Text: "Создать конфиг. Выбери режим сети:", Buttons: [][]screens.Button{
		{{Text: "С маршрутизацией", Data: "p:device:new:routed"}},
		{{Text: "Без маршрутизации", Data: "p:device:new:default"}},
		{{Text: "← Конфиги", Data: "p:devices"}},
	}}
}

func (s Service) CreateDevice(ctx context.Context, defaultNetwork bool) (screens.View, error) {
	devices, err := s.User.Devices(ctx)
	if err != nil {
		return screens.View{}, err
	}

	device, err := s.User.CreateDevice(ctx, "Конфиг "+strconv.Itoa(len(devices)+1), defaultNetwork)
	if err != nil {
		return screens.View{}, err
	}

	return deviceView(device.Name, device.URL, device.ID, device.HWID, device.DefaultNetwork), nil
}

func (s Service) Device(ctx context.Context, id string) (screens.View, error) {
	devices, err := s.User.Devices(ctx)
	if err != nil {
		return screens.View{}, err
	}

	for _, device := range devices {
		if device.ID == id {
			return deviceView(device.Name, device.URL, device.ID, device.HWID, device.DefaultNetwork), nil
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

func deviceView(name, url, id, hwid string, defaultNetwork bool) screens.View {
	status := "Ожидает первого подключения"
	if hwid != "" {
		status = "Привязано к HWID: " + html.EscapeString(hwid)
	}

	mode := "с маршрутизацией"
	if defaultNetwork {
		mode = "без маршрутизации"
	}

	text := "Конфиг «" + html.EscapeString(name) + "» (все серверы " + mode + "):\n<code>" + html.EscapeString(url) + "</code>"
	if !defaultNetwork {
		text += "\nЭкспериментальный JSON: маршрутизация на нодах и переключение VLESS-нод при отказе (Hysteria2 не входит):\n<code>" + html.EscapeString(strings.Replace(url, "/happ/device/", "/happ/device-json/", 1)) + "</code>"
	}
	return screens.View{
		Text: text + "\n" + status,
		Buttons: [][]screens.Button{
			{{Text: "Удалить устройство", Data: "p:device:delete:" + id}},
			{{Text: "← Устройства", Data: "p:devices"}},
		},
	}
}
