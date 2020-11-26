package main

type (
	Lark struct {
		api    *LarkApi
		chatId string
	}

	SendMessageArgs struct {
		Content string `json:"content"`
	}
)

func (lark *Lark) SendMessage(args *SendMessageArgs, reply *bool) (err error) {
	err = lark.api.SendMessage(lark.chatId, args.Content)
	*reply = err == nil
	return
}
