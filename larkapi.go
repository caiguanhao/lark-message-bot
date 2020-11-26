package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"time"
)

var (
	errLarkApiNotOK = errors.New("not ok returned")
)

const (
	larkApiPrefix = "https://open.feishu.cn/open-apis"
)

type (
	LarkApi struct {
		appId     string
		appSecret string

		accessToken       string
		accessTokenExpire int

		debugger func(args ...interface{})
	}

	AccessTokenResponse struct {
		Code   int    `json:"code"`
		Expire int    `json:"expire"`
		Msg    string `json:"msg"`
		Token  string `json:"tenant_access_token"`
	}

	GroupResponse struct {
		Code int `json:"code"`
		Data struct {
			ChatId string `json:"chat_id"`
		} `json:"data"`
		Msg string `json:"msg"`
	}

	MessageResponse struct {
		Code int `json:"code"`
		Data struct {
			MessageId string `json:"message_id"`
		} `json:"data"`
		Msg string `json:"msg"`
	}

	EventResponse struct {
		Type string `json:"type"`

		// type == "url_verification"
		Challenge string `json:"challenge"`

		// type == "event_callback"
		Event struct {
			Type       string `json:"type"`
			MsgType    string `json:"msg_type"`
			Text       string `json:"text"`
			UserOpenID string `json:"user_open_id"`
		} `json:"event"`
	}
)

func (lark *LarkApi) NewRequest(path string, reqBody interface{}) (resp *http.Response, err error) {
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	var reqData []byte
	reqData, err = json.Marshal(reqBody)
	if err != nil {
		return
	}
	if lark.debugger != nil {
		lark.debugger(string(reqData))
	}
	var req *http.Request
	req, err = http.NewRequest("POST", larkApiPrefix+path, bytes.NewReader(reqData))
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+lark.accessToken)
	resp, err = client.Do(req)
	return
}

func (lark *LarkApi) GetAccessToken() (err error) {
	resp, err := lark.NewRequest("/auth/v3/tenant_access_token/internal/", map[string]string{
		"app_id":     lark.appId,
		"app_secret": lark.appSecret,
	})
	if err != nil {
		return
	}
	defer resp.Body.Close()
	var res []byte
	res, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if lark.debugger != nil {
		lark.debugger(string(res))
	}
	var data AccessTokenResponse
	err = json.Unmarshal(res, &data)
	if err != nil {
		return
	}
	lark.accessToken = data.Token
	lark.accessTokenExpire = data.Expire
	return
}

func (lark *LarkApi) CreateGroup(userOpenId string) (chatId string, err error) {
	resp, err := lark.NewRequest("/chat/v4/create/", struct {
		Name    string   `json:"name"`
		OpenIds []string `json:"open_ids"`
	}{"Messages", []string{userOpenId}})
	if err != nil {
		return
	}
	defer resp.Body.Close()
	var res []byte
	res, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if lark.debugger != nil {
		lark.debugger(string(res))
	}
	var data GroupResponse
	err = json.Unmarshal(res, &data)
	if err != nil {
		return
	}
	if data.Msg == "ok" {
		chatId = data.Data.ChatId
	} else {
		err = errLarkApiNotOK
	}
	return
}

func (lark *LarkApi) AddUserToChat(userOpenId, chatId string) (err error) {
	resp, err := lark.NewRequest("/chat/v4/chatter/add/", struct {
		ChatId  string   `json:"chat_id"`
		OpenIDs []string `json:"open_ids"`
	}{chatId, []string{userOpenId}})
	if err != nil {
		return
	}
	defer resp.Body.Close()
	var res []byte
	res, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if lark.debugger != nil {
		lark.debugger(string(res))
	}
	var data GroupResponse
	err = json.Unmarshal(res, &data)
	if err != nil {
		return
	}
	if data.Msg != "ok" {
		err = errLarkApiNotOK
	}
	return
}

func (lark *LarkApi) SendMessage(chatId, content string) (err error) {
	resp, err := lark.NewRequest("/message/v4/send/", struct {
		ChatId  string      `json:"chat_id"`
		MsgType string      `json:"msg_type"`
		Content interface{} `json:"content"`
	}{chatId, "text", struct {
		Text string `json:"text"`
	}{content}})
	if err != nil {
		return
	}
	defer resp.Body.Close()
	var res []byte
	res, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if lark.debugger != nil {
		lark.debugger(string(res))
	}
	var data MessageResponse
	err = json.Unmarshal(res, &data)
	if err != nil {
		return
	}
	if data.Msg != "ok" {
		err = errLarkApiNotOK
	}
	return
}
