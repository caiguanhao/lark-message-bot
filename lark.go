package main

import (
	"errors"

	"github.com/caiguanhao/lark-slim"
)

type (
	Lark struct {
		api *lark.API
	}

	SendMessageArgs struct {
		ChatId  string `json:"chat_id"`
		Content string `json:"content"`
	}

	SendPostArgs struct {
		ChatId string    `json:"chat_id"`
		Post   lark.Post `json:"post"`
	}
)

func (lark *Lark) SendMessage(args *SendMessageArgs, reply *bool) (err error) {
	chatId := args.ChatId
	if chatId == "" {
		err = errors.New("please specify chat id")
	} else {
		err = lark.api.SendMessage(chatId, args.Content)
	}
	*reply = err == nil
	return
}

func (lark *Lark) SendPost(args *SendPostArgs, reply *bool) (err error) {
	chatId := args.ChatId
	if chatId == "" {
		err = errors.New("please specify chat id")
	} else {
		err = lark.api.SendPost(chatId, args.Post)
	}
	*reply = err == nil
	return
}
