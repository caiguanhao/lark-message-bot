package main

import (
	"flag"
)

func main() {
	appId := flag.String("app-id", "", "lark app id (you can also use env LARK_APP_ID)")
	appSecret := flag.String("app-secret", "", "lark app secret (you can also use env LARK_APP_SECRET)")
	address := flag.String("listen", "127.0.0.1:32123", "private http server address")
	verbosity := flag.String("verbosity", "info", "debug, info, notice, warning, error, critical")
	masters := flag.String("masters", "", `user ids to work for, separated by ","`)
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
		masters: *masters,
	}
	h.init().serve(*address)
}
