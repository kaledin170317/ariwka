package main

import (
	"context"
	"log"
	"os"

	"vpn/internal/telegram/bot"
)

func main() {
	err := bot.Run(context.Background(), os.Getenv("TELEGRAM_BOT_TOKEN"), os.Getenv("TELEGRAM_CONTROL_USERNAME"),
		os.Getenv("TELEGRAM_CONTROL_PASSWORD"), os.Getenv("CONTROL_ADDRESS"))
	if err != nil {
		log.Fatal(err)
	}
}
