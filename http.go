package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"strings"

	"github.com/caiguanhao/larkslim"
)

type (
	httpHandler struct {
		lark *Lark

		aliyun *aliyunClient

		masters     string
		mastersList []string
	}
)

func (_h *httpHandler) init() (h *httpHandler) {
	h = _h

	if h.lark.api.AppId == "" {
		h.lark.api.AppId = os.Getenv("LARK_APP_ID")
	}
	if h.lark.api.AppId == "" {
		log.Fatal("error: empty app id")
	}

	if h.lark.api.AppSecret == "" {
		h.lark.api.AppSecret = os.Getenv("LARK_APP_SECRET")
	}
	if h.lark.api.AppSecret == "" {
		log.Fatal("error: empty app secret")
	}

	if h.aliyun == nil {
		h.aliyun = &aliyunClient{}
	}
	if h.aliyun.AccessKeyId == "" {
		h.aliyun.AccessKeyId = os.Getenv("ALIYUN_ACCESS_KEY_ID")
	}
	if h.aliyun.AccessKeySecret == "" {
		h.aliyun.AccessKeySecret = os.Getenv("ALIYUN_ACCESS_KEY_SECRET")
	}
	if h.aliyun.MainDomain == "" {
		h.aliyun.MainDomain = os.Getenv("ALIYUN_MAIN_DOMAIN")
	}
	if h.aliyun.DefaultIP == "" {
		h.aliyun.DefaultIP = os.Getenv("ALIYUN_DEFAULT_IP")
	}
	if h.aliyun.configured() {
		log.Notice("aliyun dns enabled for domain", h.aliyun.MainDomain)
	}

	masters := strings.Split(h.masters, ",")
	for _, id := range masters {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		h.mastersList = append(h.mastersList, id)
	}
	if len(h.mastersList) > 0 {
		log.Noticef("serving for %d users: %s", len(h.mastersList), h.mastersList)
	} else {
		log.Notice("serving for all users")
	}

	err := rpc.Register(h.lark)
	if err != nil {
		log.Fatal("json-rpc error:", err)
	}

	return
}

func (h *httpHandler) serve(address string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/jsonrpc/", h.handleJSONRPC)
	mux.HandleFunc("/", h.handle404)
	server := &http.Server{
		Addr:    address,
		Handler: h.logRequest(mux),
	}
	log.Info("listening", address)
	log.Fatal(server.ListenAndServe())
}

func (h *httpHandler) handleJSONRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}
	if r.Body == nil {
		http.NotFound(w, r)
		return
	}
	defer r.Body.Close()
	res := JSONRPCRequest{r.Body, &bytes.Buffer{}, make(chan bool)}
	codec := Codec{codec: jsonrpc.NewServerCodec(&res)}
	go rpc.ServeCodec(&codec)
	<-res.done
	w.Header().Set("Content-Type", "application/json")
	if codec.isError {
		w.WriteHeader(400)
	}
	_, err := io.Copy(w, res.readWriter)
	if err != nil {
		log.Error("response error:", err)
		return
	}
}

func (h *httpHandler) handleMessagePayload(payload []byte) error {
	var event larkslim.MessageReceiveEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return err
	}
	if event.Header.EventType != "im.message.receive_v1" {
		return nil
	}
	h.handleMessageReceive(event)
	return nil
}

func (h *httpHandler) handleMessageReceive(event larkslim.MessageReceiveEvent) {
	// ignore messages sent by bots (including itself) to avoid loops
	if event.Event.Sender.SenderType != "user" {
		return
	}
	log.Info("received", event.Event.Message.MessageType, "message:", event.Event.Message.Content)
	if event.Event.Message.MessageType != "text" {
		return
	}
	input := strings.TrimSpace(event.TextWithoutMentions())
	if input == "" {
		input = strings.TrimSpace(event.Text())
	}
	log.Info("message input:", input)
	if input == "" {
		return
	}

	senderOpenId := event.Event.Sender.SenderId.OpenId
	chatId := event.Event.Message.ChatId
	if h.isMaster(senderOpenId) {
		h.reply(chatId, call(MastersCall{
			Call{
				senderOpenId: senderOpenId,
				http:         h,
			},
		}, input))
	} else {
		h.reply(chatId, call(Call{
			senderOpenId: senderOpenId,
			http:         h,
		}, input))
	}
}

func (h *httpHandler) handle404(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}

func (h *httpHandler) logRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Debug(r.Method, r.URL)
		handler.ServeHTTP(w, r)
	})
}

func (h *httpHandler) reply(chatId, message string) {
	err := h.lark.api.SendMessage(chatId, message)
	if err != nil {
		log.Error(err)
	}
}

func (h *httpHandler) isMaster(userId string) bool {
	if len(h.mastersList) == 0 {
		return true
	}
	for _, id := range h.mastersList {
		if userId == id {
			return true
		}
	}
	return false
}
