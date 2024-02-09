package main

import (
	"context"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

func startTaskBot(ctx context.Context, httpListenAddr string) error {
	// сюда писать код
	/*
		в этом месте вы стартуете бота,
		стартуете хттп сервер который будет обслуживать этого бота
		инициализируете ваше приложение
		и потом будете обрабатывать входящие сообщения
	*/
	return nil
}

func main() {
	err := startTaskBot(context.Background(), ":8081")
	if err != nil {
		log.Fatalln(err)
	}
}

//  это заглушка чтобы импорт сохранился
func __dummy() {
	tgbotapi.APIEndpoint = "_dummy"
}
