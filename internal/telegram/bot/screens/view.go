package screens

type Button struct {
	Text string `json:"text"`
	Data string `json:"callback_data"`
}

type View struct {
	Text    string
	Buttons [][]Button
}

func Home() View {
	return View{
		Text: "Главная. Выбери раздел:",
		Buttons: [][]Button{
			{{Text: "Информация о проекте", Data: "project"}},
			{{Text: "Профили", Data: "profiles"}},
		},
	}
}
