package main

import (
	"context"
	"log"

	"github.com/golang-vk-backend-challenge/subpub"
)

func main() {
	sp := subpub.NewSubPub()
	cb := func(msg any) {
		log.Println("I have got a msg:", msg)
	}
	sub, _ := sp.Subscribe("test", cb)

	sub.WG.Add(1)
	go func() {
		defer sub.WG.Done()
		for msg := range sub.Queue {
			sub.Callback(msg)
		}
	}()

	if err := sp.Publish("test", "First Test MSG!!!"); err != nil {
		log.Fatalln(err)
	}

	if err := sp.Close(context.Background()); err != nil {
		log.Fatalln(err)
	}
}
