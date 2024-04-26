package main

import (
	"bot/internal"
	"flag"
	"log"
)

func main() {
	token := flag.String("BOT_TOKEN", "", "")
	flag.Parse()

	client, err := internal.NewClient(*token, false, true)
	if err != nil {
		log.Fatal(err)
	}

	err = client.Run()
	if err != nil {
		log.Fatal(err)
	}
}
