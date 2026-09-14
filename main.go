package main

import (
	"context"
	"flag"

	"github.com/caiguanhao/larkslim"
	"github.com/caiguanhao/larkslim/ws"
)

func main() {
	appId := flag.String("app-id", "", "lark app id (you can also use env LARK_APP_ID)")
	appSecret := flag.String("app-secret", "", "lark app secret (you can also use env LARK_APP_SECRET)")
	address := flag.String("listen", "127.0.0.1:32123", "private http server address")
	verbosity := flag.String("verbosity", "info", "debug, info, notice, warning, error, critical")
	masters := flag.String("masters", "", `user ids to work for, separated by ","`)
	aliyunAccessKeyId := flag.String("aliyun-access-key-id", "", "aliyun access key id (you can also use env ALIYUN_ACCESS_KEY_ID)")
	aliyunAccessKeySecret := flag.String("aliyun-access-key-secret", "", "aliyun access key secret (you can also use env ALIYUN_ACCESS_KEY_SECRET)")
	aliyunMainDomain := flag.String("aliyun-main-domain", "", "main domain for create_subdomain command (you can also use env ALIYUN_MAIN_DOMAIN)")
	aliyunDefaultIP := flag.String("aliyun-default-ip", "", "default A record IP for create_subdomain command (you can also use env ALIYUN_DEFAULT_IP)")
	flag.Parse()

	initLogger(*verbosity)

	h := &httpHandler{
		lark: &Lark{
			api: &larkslim.API{
				AppId:     *appId,
				AppSecret: *appSecret,
				Debugger:  log.Debug,
			},
		},
		aliyun: &aliyunClient{
			AccessKeyId:     *aliyunAccessKeyId,
			AccessKeySecret: *aliyunAccessKeySecret,
			MainDomain:      *aliyunMainDomain,
			DefaultIP:       *aliyunDefaultIP,
		},
		masters: *masters,
	}
	h.init()
	go h.serve(*address)

	client := ws.NewClient(h.lark.api.AppId, h.lark.api.AppSecret,
		ws.WithEventHandler(func(_ context.Context, payload []byte) error {
			return h.handleMessagePayload(payload)
		}),
	)
	log.Fatal(client.Start(context.Background()))
}
