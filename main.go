package main

import (
	"flag"
)

func main() {
	appId := flag.String("app-id", "", "lark app id (you can also use env LARK_APP_ID)")
	appSecret := flag.String("app-secret", "", "lark app secret (you can also use env LARK_APP_SECRET)")
	chatIdFile := flag.String("chat-id", "chat.id", "file to store chat id")
	address := flag.String("listen", "127.0.0.1:32123", "private http server address")
	verbosity := flag.String("verbosity", "info", "debug, info, notice, warning, error, critical")
	flag.Parse()

	initLogger(*verbosity)

	h := &httpHandler{
		lark: &Lark{
			api: &LarkApi{
				appId:     *appId,
				appSecret: *appSecret,
				debugger:  log.Debug,
			},
		},
		chatIdFile: *chatIdFile,
	}
	h.init().serve(*address)
}
