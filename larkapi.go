package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
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

	LarkApiResponse struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}

	AccessTokenResponse struct {
		LarkApiResponse
		Expire int    `json:"expire"`
		Token  string `json:"tenant_access_token"`
	}

	GroupResponse struct {
		LarkApiResponse
		Data struct {
			ChatId string `json:"chat_id"`
		} `json:"data"`
	}

	Group struct {
		Avatar      string `json:"avatar"`
		ChatId      string `json:"chat_id"`
		Description string `json:"description"`
		Name        string `json:"name"`
		OwnerOpenId string `json:"owner_open_id"`
		OwnerUserId string `json:"owner_user_id"`
	}

	Groups []Group

	GroupsResponse struct {
		LarkApiResponse
		Data struct {
			Groups Groups `json:"groups"`
		} `json:"data"`
	}

	MessageResponse struct {
		LarkApiResponse
		Data struct {
			MessageId string `json:"message_id"`
		} `json:"data"`
	}

	EventResponse struct {
		Type string `json:"type"`

		// type == "url_verification"
		Challenge string `json:"challenge"`

		// type == "event_callback"
		Event struct {
			ChatId     string `json:"open_chat_id"`
			Type       string `json:"type"`
			MsgType    string `json:"msg_type"`
			Text       string `json:"text"`
			UserOpenID string `json:"user_open_id"`
		} `json:"event"`
	}

	PostTag struct {
		Tag      string `json:"tag"`
		Unescape bool   `json:"un_escape"`
		Text     string `json:"text"`
		Href     string `json:"href"`
		UserId   string `json:"user_id"`
		ImageKey string `json:"image_key"`
		Width    int    `json:"width"`
		Height   int    `json:"height"`
	}

	PostLine []PostTag

	PostLines []PostLine

	PostOfLocale struct {
		Title   string    `json:"title"`
		Content PostLines `json:"content"`
	}

	Post map[string]PostOfLocale
)

func (lark *LarkApi) NewRequest(path string, reqBody interface{}, respData interface{}) (err error) {
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
	var resp *http.Response
	resp, err = client.Do(req)
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
	var apiResp LarkApiResponse
	err = json.Unmarshal(res, &apiResp)
	if err != nil {
		return
	}
	if apiResp.Msg != "ok" {
		err = fmt.Errorf("not ok returned: %s", apiResp.Msg)
		return
	}
	if respData != nil {
		err = json.Unmarshal(res, respData)
	}
	return
}

func (lark *LarkApi) GetAccessToken() (err error) {
	var data AccessTokenResponse
	err = lark.NewRequest(
		// path
		"/auth/v3/tenant_access_token/internal/",

		// request body
		map[string]string{
			"app_id":     lark.appId,
			"app_secret": lark.appSecret,
		},

		// response
		&data,
	)
	if err != nil {
		return
	}
	lark.accessToken = data.Token
	lark.accessTokenExpire = data.Expire
	return
}

func (lark *LarkApi) ListAllChats() (groups Groups, err error) {
	var data GroupsResponse
	err = lark.NewRequest(
		// path
		"/chat/v4/list/",

		// request body
		struct {
			PageSize string `json:"page_size"`
		}{"200"},

		// response
		&data,
	)
	groups = data.Data.Groups
	return
}

func (lark *LarkApi) CreateChat(name, userOpenId string) (chatId string, err error) {
	var data GroupResponse
	err = lark.NewRequest(
		// path
		"/chat/v4/create/",

		// request body
		struct {
			Name    string   `json:"name"`
			OpenIds []string `json:"open_ids"`
		}{name, []string{userOpenId}},

		// response
		&data,
	)
	if err != nil {
		return
	}
	chatId = data.Data.ChatId
	return
}

func (lark *LarkApi) DestroyChat(chatId string) (err error) {
	err = lark.NewRequest(
		// path
		"/chat/v4/disband/",

		// request body
		struct {
			ChatId string `json:"chat_id"`
		}{chatId},

		// response
		nil,
	)
	return
}

func (lark *LarkApi) AddUserToChat(userOpenId, chatId string) (err error) {
	var data GroupResponse
	err = lark.NewRequest(
		// path
		"/chat/v4/chatter/add/",

		// request body
		struct {
			ChatId  string   `json:"chat_id"`
			OpenIDs []string `json:"open_ids"`
		}{chatId, []string{userOpenId}},

		// response
		&data,
	)
	if err != nil {
		return
	}
	return
}

func (lark *LarkApi) SendMessage(chatId, content string) (err error) {
	var data MessageResponse
	err = lark.NewRequest(
		// path
		"/message/v4/send/",

		// request body
		struct {
			ChatId  string      `json:"chat_id"`
			MsgType string      `json:"msg_type"`
			Content interface{} `json:"content"`
		}{chatId, "text", struct {
			Text string `json:"text"`
		}{content}},

		// response
		&data,
	)
	return
}

func (lark *LarkApi) SendPost(chatId string, post Post) (err error) {
	var data MessageResponse
	err = lark.NewRequest(
		// path
		"/message/v4/send/",

		// request body
		struct {
			ChatId  string      `json:"chat_id"`
			MsgType string      `json:"msg_type"`
			Content interface{} `json:"content"`
		}{chatId, "post", struct {
			Post Post `json:"post"`
		}{post}},

		// response
		&data,
	)
	return
}

func (groups *Groups) String() string {
	if len(*groups) == 0 {
		return "no groups"
	}
	var b bytes.Buffer
	for i, group := range *groups {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(fmt.Sprintf("%d. ", i+1))
		b.WriteString(group.Name)
		b.WriteString(": ")
		b.WriteString(group.ChatId)
	}
	return b.String()
}
