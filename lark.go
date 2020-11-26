package main

type (
	Lark struct {
		api    *LarkApi
		chatId string
	}

	SendMessageArgs struct {
		Content string `json:"content"`
	}

	SendPostArgs struct {
		Post Post `json:"post"`
	}
)

func (lark *Lark) SendMessage(args *SendMessageArgs, reply *bool) (err error) {
	err = lark.api.SendMessage(lark.chatId, args.Content)
	*reply = err == nil
	return
}

func (lark *Lark) SendPost(args *SendPostArgs, reply *bool) (err error) {
	err = lark.api.SendPost(lark.chatId, args.Post)
	*reply = err == nil
	return
}
