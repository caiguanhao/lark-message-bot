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
)

type (
	httpHandler struct {
		lark *Lark

		chatIdFile string
	}
)

func (_h *httpHandler) init() (h *httpHandler) {
	h = _h

	if h.lark.api.appId == "" {
		h.lark.api.appId = os.Getenv("LARK_APP_ID")
	}
	if h.lark.api.appId == "" {
		log.Fatal("error: empty app id")
	}

	if h.lark.api.appSecret == "" {
		h.lark.api.appSecret = os.Getenv("LARK_APP_SECRET")
	}
	if h.lark.api.appSecret == "" {
		log.Fatal("error: empty app secret")
	}

	err := rpc.Register(h.lark)
	if err != nil {
		log.Fatal("json-rpc error:", err)
	}

	go h.updateAccessToken()

	chatId, err := ioutil.ReadFile(h.chatIdFile)
	if err == nil {
		h.lark.chatId = strings.TrimSpace(string(chatId))
		log.Notice("using chat id", h.lark.chatId)
	}
	if h.lark.chatId == "" {
		log.Notice("no chat id: will be created in the future")
	}
	return
}

func (h *httpHandler) serve(address string) {
	http.HandleFunc("/jsonrpc/", h.handleJSONRPC)
	http.HandleFunc("/", h.handleLarkEvents)
	log.Fatal(http.ListenAndServe(address, nil))
}

func (h *httpHandler) updateAccessToken() {
	defer func() {
		time.Sleep(5 * time.Second)
		h.lark.api.GetAccessToken()
	}()
	h.lark.api.GetAccessToken()
	secs := h.lark.api.accessTokenExpire - 60
	if secs < 5 {
		secs = 5
	}
	log.Info("access token has changed to", h.lark.api.accessToken, "next in", secs, "seconds")
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
	var resp EventResponse
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
		if resp.Event.Type == "message" && resp.Event.MsgType == "text" && resp.Event.Text == "+" {
			if h.lark.chatId == "" {
				chatId, err := h.lark.api.CreateGroup(resp.Event.UserOpenID)
				if err == nil {
					h.lark.chatId = chatId
					err = ioutil.WriteFile(h.chatIdFile, []byte(h.lark.chatId), 0644)
					if err != nil {
						log.Error(err)
					}
				} else {
					log.Error(err)
				}

			} else {
				h.lark.api.AddUserToChat(resp.Event.UserOpenID, h.lark.chatId)
			}
		}
	}
	w.WriteHeader(http.StatusNoContent)
}
