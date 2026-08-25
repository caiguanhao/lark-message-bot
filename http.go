package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/caiguanhao/larkslim"
)

type (
	httpHandler struct {
		lark *Lark

		masters     string
		mastersList []string

		seenMutex    sync.Mutex
		seenMessages map[string]time.Time
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

	h.seenMessages = map[string]time.Time{}

	return
}

func (h *httpHandler) serve(address string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/jsonrpc/", h.handleJSONRPC)
	mux.HandleFunc("/events/", h.handleLarkEvents)
	mux.HandleFunc("/204/", h.handle204)
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
	return
}

func (h *httpHandler) handleLarkEvents(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}
	log.Debug(string(body))
	var verification struct {
		Type      string `json:"type"`
		Challenge string `json:"challenge"`
	}
	if json.Unmarshal(body, &verification) == nil &&
		verification.Type == "url_verification" {
		if data, err := json.Marshal(map[string]string{
			"challenge": verification.Challenge,
		}); err == nil {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, string(data))
			return
		}
	}
	var event larkslim.MessageReceiveEvent
	if json.Unmarshal(body, &event) != nil ||
		event.Header.EventType != "im.message.receive_v1" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	// feishu re-pushes the event if the callback is not acknowledged in
	// time, so respond first and handle the message asynchronously
	if h.isDuplicate(event.Event.Message.MessageId) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	go h.handleMessageReceive(event)
	w.WriteHeader(http.StatusNoContent)
}

func (h *httpHandler) handleMessageReceive(event larkslim.MessageReceiveEvent) {
	// ignore messages sent by bots (including itself) to avoid loops
	if event.Event.Sender.SenderType != "user" {
		return
	}
	if event.Event.Message.MessageType != "text" {
		return
	}
	input := strings.TrimSpace(event.TextWithoutMentions())
	if input == "" {
		input = strings.TrimSpace(event.Text())
	}
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

func (h *httpHandler) handle204(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

// isDuplicate records message id and reports whether it has been seen
// before. Feishu re-pushes an event if the callback is not acknowledged
// fast enough, so the same message may be delivered more than once.
func (h *httpHandler) isDuplicate(messageId string) bool {
	if messageId == "" {
		return false
	}
	h.seenMutex.Lock()
	defer h.seenMutex.Unlock()
	if _, ok := h.seenMessages[messageId]; ok {
		return true
	}
	h.seenMessages[messageId] = time.Now()
	for id, t := range h.seenMessages {
		if time.Since(t) > 10*time.Minute {
			delete(h.seenMessages, id)
		}
	}
	return false
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
