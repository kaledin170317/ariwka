package profiles

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"html"

	"vpn/internal/telegram/bot/screens"
)

func (s Service) NodesList(ctx context.Context) (screens.View, error) {
	nodes, err := s.User.Nodes(ctx)
	if err != nil {
		return screens.View{}, err
	}

	rows := make([][]screens.Button, 0, len(nodes)+1)
	for _, node := range nodes {
		data := "p:node:" + node.ID
		if len(data) <= 64 {
			rows = append(rows, []screens.Button{{Text: node.Name, Data: data}})
		}
	}

	rows = append(rows, []screens.Button{{Text: "← Профили", Data: "profiles"}})
	if len(rows) == 1 {
		return screens.View{Text: "Доступных нод пока нет.", Buttons: rows}, nil
	}

	return screens.View{Text: "Создать профиль. Выбери ноду:", Buttons: rows}, nil
}

func (s Service) Inbounds(ctx context.Context, nodeID string) (screens.View, error) {
	inbounds, err := s.User.Inbounds(ctx, nodeID)
	if err != nil {
		return screens.View{}, err
	}

	rows := make([][]screens.Button, 0, len(inbounds)+1)
	for _, inbound := range inbounds {
		data := "p:new:" + nodeID + ":" + inboundKey(inbound.Tag)
		if len(data) <= 64 {
			rows = append(rows, []screens.Button{{Text: inbound.Tag + " (" + inbound.Type + ")", Data: data}})
		}
	}

	rows = append(rows, []screens.Button{{Text: "← К нодам", Data: "p:create"}})
	if len(rows) == 1 {
		return screens.View{Text: "На этой ноде пока нет inbound.", Buttons: rows}, nil
	}

	return screens.View{Text: "Выбери inbound:", Buttons: rows}, nil
}

func (s Service) Create(ctx context.Context, nodeID, key string) (screens.View, error) {
	inbounds, err := s.User.Inbounds(ctx, nodeID)
	if err != nil {
		return screens.View{}, err
	}

	for _, inbound := range inbounds {
		if inboundKey(inbound.Tag) == key {
			connection, err := s.User.CreateProfile(ctx, nodeID, inbound.Tag)
			if err != nil {
				return screens.View{}, err
			}

			return screens.View{
				Text:    "Профиль создан:\n<code>" + html.EscapeString(connection) + "</code>",
				Buttons: [][]screens.Button{{{Text: "← Профили", Data: "profiles"}}},
			}, nil
		}
	}

	return screens.View{Text: "Inbound больше недоступен. Выбери ноду заново.", Buttons: [][]screens.Button{{{Text: "← К нодам", Data: "p:create"}}}}, nil
}

func inboundKey(tag string) string {
	sum := sha256.Sum256([]byte(tag))
	return hex.EncodeToString(sum[:8])
}
