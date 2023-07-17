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

	"github.com/caiguanhao/larkslim"
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
	var resp larkslim.EventResponse
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

func (h *httpHandler) handleEventCallback(resp larkslim.EventResponse) {
	if resp.Event.Type != "message" || resp.Event.MsgType != "text" {
		return
	}
	input := strings.TrimSpace(resp.Event.TextWithoutAtBot)
	if input == "" {
		input = strings.TrimSpace(resp.Event.Text)
	}

	if h.isMaster(resp.Event.OpenId) {
		h.reply(resp.Event.ChatId, call(MastersCall{
			Call{
				resp: resp,
				http: h,
			},
		}, input))
	} else {
		h.reply(resp.Event.ChatId, call(Call{
			resp: resp,
			http: h,
		}, input))
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
