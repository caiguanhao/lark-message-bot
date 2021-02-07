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
	"time"

	"github.com/caiguanhao/lark-slim"
)

type (
	httpHandler struct {
		lark *Lark

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

	go h.updateAccessToken()

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

func (h *httpHandler) updateAccessToken() {
	defer func() {
		time.Sleep(5 * time.Second)
		h.updateAccessToken()
	}()
	expire, err := h.lark.api.GetAccessToken()
	if err != nil {
		log.Error(err)
		return
	}
	secs := expire - 60
	if secs < 5 {
		secs = 5
	}
	log.Info("update access token in", secs, "seconds")
	time.Sleep(time.Duration(secs) * time.Second)
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
	var resp lark.EventResponse
	if json.Unmarshal(body, &resp) != nil {
		return
	}
	log.Debug(string(body))
	switch resp.Type {
	case "url_verification":
		if data, err := json.Marshal(map[string]string{
			"challenge": resp.Challenge,
		}); err == nil {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, string(data))
			return
		}
	case "event_callback":
		h.handleEventCallback(resp)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *httpHandler) handleEventCallback(resp lark.EventResponse) {
	if resp.Event.Type != "message" || resp.Event.MsgType != "text" {
		return
	}
	text := strings.TrimSpace(resp.Event.TextWithoutAtBot)
	if text == "" {
		text = strings.TrimSpace(resp.Event.Text)
	}
	funcName, args := h.parseCall(text)
	if funcName == "whoami" {
		h.reply(resp.Event.ChatId, resp.Event.OpenId)
		return
	}
	if funcName == "whosyourdaddy" {
		h.reply(resp.Event.ChatId, "CGH")
		return
	}
	if !h.isAllowed(resp.Event.OpenId) {
		h.reply(resp.Event.ChatId, "sorry, I am not allowed to work for you")
		return
	}
	if funcName == "create" {
		if len(args) == 0 || args[0] == "" {
			h.reply(resp.Event.ChatId, "name is needed to create a chat, specify like this:\ncreate(name)")
			return
		}
		chatId, err := h.lark.api.CreateChat(args[0], resp.Event.UserOpenId)
		if err != nil {
			log.Error(err)
			h.reply(resp.Event.ChatId, err.Error())
			return
		}
		h.reply(resp.Event.ChatId, fmt.Sprintf(`chat with name "%s" has been created, its id is %s`, args[0], chatId))
		return
	}
	if funcName == "join" {
		if len(args) == 0 || args[0] == "" {
			h.reply(resp.Event.ChatId, "chat id is needed to join a chat, specify like this:\njoin(chat-id...)")
			return
		}
		for _, chatId := range args {
			err := h.lark.api.AddUsersToChat(chatId, []string{resp.Event.UserOpenId})
			if err != nil {
				log.Error(err)
				h.reply(resp.Event.ChatId, err.Error())
				return
			}
			h.reply(resp.Event.ChatId, fmt.Sprintf("successfully joined chat: %s", chatId))
		}
		return
	}
	if funcName == "destroy" {
		if len(args) == 0 || args[0] == "" {
			h.reply(resp.Event.ChatId, "chat id is needed to destroy a chat, specify like this:\ndestroy(chat-id...)")
			return
		}
		for _, chatId := range args {
			err := h.lark.api.DestroyChat(chatId)
			if err != nil {
				log.Error(err)
				h.reply(resp.Event.ChatId, err.Error())
				return
			}
			h.reply(resp.Event.ChatId, fmt.Sprintf("successfully destroyed chat: %s", chatId))
		}
		return
	}
	if funcName == "members" {
		if len(args) == 0 || args[0] == "" {
			h.reply(resp.Event.ChatId, "chat id is needed to list members of a chat, specify like this:\nmembers(chat-id)")
			return
		}
		group, err := h.lark.api.GetChatInfo(args[0])
		if err != nil {
			log.Error(err)
			h.reply(resp.Event.ChatId, err.Error())
			return
		}
		userIds := []string{}
		for _, m := range group.Members {
			userIds = append(userIds, m.OpenId)
		}
		userInfos, err := h.lark.api.GetUserInfo(userIds)
		if err != nil {
			log.Error(err)
			h.reply(resp.Event.ChatId, err.Error())
			return
		}
		h.reply(resp.Event.ChatId, userInfos.String())
		return
	}
	if funcName == "add" {
		if len(args) < 2 || args[0] == "" {
			h.reply(resp.Event.ChatId, "chat id is needed to add users to a chat, specify like this:\nadd(chat-id, user-id...)")
			return
		}
		err := h.lark.api.AddUsersToChat(args[0], args[1:])
		if err != nil {
			log.Error(err)
			h.reply(resp.Event.ChatId, err.Error())
			return
		}
		h.reply(resp.Event.ChatId, "successfully added users to chat")
		return
	}
	if funcName == "remove" {
		if len(args) < 2 || args[0] == "" {
			h.reply(resp.Event.ChatId, "chat id is needed to remove users from a chat, specify like this:\nremove(chat-id, user-id...)")
			return
		}
		err := h.lark.api.RemoveUsersFromChat(args[0], args[1:])
		if err != nil {
			log.Error(err)
			h.reply(resp.Event.ChatId, err.Error())
			return
		}
		h.reply(resp.Event.ChatId, "successfully removed users from chat")
		return
	}
	if funcName == "list" {
		groups, err := h.lark.api.ListAllChats()
		if err != nil {
			log.Error(err)
			h.reply(resp.Event.ChatId, err.Error())
			return
		}
		h.reply(resp.Event.ChatId, groups.String())
		return
	}
	err := h.lark.api.SendMessage(resp.Event.ChatId, "unknown function")
	if err != nil {
		log.Error(err)
	}
}

func (h *httpHandler) handle204(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
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

func (h *httpHandler) parseCall(input string) (funcName string, args []string) {
	input = strings.TrimSpace(input)
	if len(input) < 1 {
		return
	}
	start := strings.Index(input, "(")
	if start == -1 {
		funcName = input
		return
	}
	funcName = strings.TrimSpace(input[0:start])
	if input[len(input)-1] != ')' {
		return
	}
	args = strings.Split(input[start+1:len(input)-1], ",")
	for i := range args {
		args[i] = strings.TrimSpace(args[i])
	}
	if len(args) == 1 && args[0] == "" {
		args = []string{}
	}
	return
}

func (h *httpHandler) reply(chatId, message string) {
	err := h.lark.api.SendMessage(chatId, message)
	if err != nil {
		log.Error(err)
	}
}

func (h *httpHandler) isAllowed(userId string) bool {
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
