package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"reflect"
	"strings"

	"github.com/caiguanhao/lark-slim"
)

type (
	Call struct {
		resp lark.EventResponse
		http *httpHandler
	}

	MastersCall struct {
		Call
	}
)

func (c Call) WhoAmI() string {
	return c.resp.Event.OpenId
}

func (c Call) WhosYourDaddy() string {
	userInfos, err := c.http.lark.api.GetUserInfo(c.http.mastersList)
	if err != nil {
		log.Error(err)
		return err.Error()
	}
	return userInfos.String()
}

func (c MastersCall) Create(names ...string) string {
	if len(names) == 0 || names[0] == "" {
		return "name is needed to create a chat, specify like this:\ncreate(name)"
	}
	chatId, err := c.http.lark.api.CreateChat(names[0], c.resp.Event.UserOpenId)
	if err != nil {
		log.Error(err)
		return err.Error()
	}
	return fmt.Sprintf(`chat with name "%s" has been created, its id is %s`, names[0], chatId)
}

func (c MastersCall) Join(chatIds ...string) string {
	if len(chatIds) == 0 || chatIds[0] == "" {
		return "chat id is needed to join a chat, specify like this:\njoin(chat-id...)"
	}
	ret := []string{}
	for _, chatId := range chatIds {
		err := c.http.lark.api.AddUsersToChat(chatId, []string{c.resp.Event.UserOpenId})
		if err != nil {
			log.Error(err)
			ret = append(ret, err.Error())
			continue
		}
		ret = append(ret, fmt.Sprintf("successfully joined chat: %s", chatId))
	}
	return strings.Join(ret, "\n")
}

func (c MastersCall) Destroy(chatIds ...string) string {
	if len(chatIds) == 0 || chatIds[0] == "" {
		return "chat id is needed to destroy a chat, specify like this:\ndestroy(chat-id...)"
	}
	ret := []string{}
	for _, chatId := range chatIds {
		err := c.http.lark.api.DestroyChat(chatId)
		if err != nil {
			log.Error(err)
			ret = append(ret, err.Error())
			continue
		}
		ret = append(ret, fmt.Sprintf("successfully destroyed chat: %s", chatId))
	}
	return strings.Join(ret, "\n")
}

func (c MastersCall) Members(chatIds ...string) string {
	if len(chatIds) == 0 || chatIds[0] == "" {
		return "chat id is needed to list members of a chat, specify like this:\nmembers(chat-id)"
	}
	group, err := c.http.lark.api.GetChatInfo(chatIds[0])
	if err != nil {
		log.Error(err)
		return err.Error()
	}
	userIds := []string{}
	for _, m := range group.Members {
		userIds = append(userIds, m.OpenId)
	}
	userInfos, err := c.http.lark.api.GetUserInfo(userIds)
	if err != nil {
		log.Error(err)
		return err.Error()
	}
	return userInfos.String()
}

func (c MastersCall) Add(ids ...string) string {
	if len(ids) < 2 || ids[0] == "" {
		return "chat id is needed to add users to a chat, specify like this:\nadd(chat-id, user-id...)"
	}
	err := c.http.lark.api.AddUsersToChat(ids[0], ids[1:])
	if err != nil {
		log.Error(err)
		return err.Error()
	}
	return "successfully added users to chat"
}

func (c MastersCall) Remove(ids ...string) string {
	if len(ids) < 2 || ids[0] == "" {
		return "chat id is needed to remove users from a chat, specify like this:\nremove(chat-id, user-id...)"
	}
	err := c.http.lark.api.RemoveUsersFromChat(ids[0], ids[1:])
	if err != nil {
		log.Error(err)
		return err.Error()
	}
	return "successfully removed users from chat"
}

func (c MastersCall) List() string {
	groups, err := c.http.lark.api.ListAllChats()
	if err != nil {
		log.Error(err)
		return err.Error()
	}
	return groups.String()
}

const (
	callUnknownExpression = "unknown expression"
	callUnknownFunction   = "unknown function"
	callTooFewArguments   = "too few arguments"
	callTooManyArguments  = "too many arguments"
)

func call(obj interface{}, expression string) string {
	expr, err := parser.ParseExpr(expression)
	if err != nil {
		return callUnknownExpression
	}

	var method string
	var args []string

	switch e := expr.(type) {
	case *ast.CallExpr:
		method = expression[e.Fun.Pos()-1 : e.Fun.End()-1]
		for _, a := range e.Args {
			args = append(args, expression[a.Pos()-1:a.End()-1])
		}
	case *ast.Ident:
		method = expression[e.Pos()-1 : e.End()-1]
		// no args
	default:
		return callUnknownExpression
	}
	method = strings.ToLower(method)

	rt := reflect.TypeOf(obj)
	methodNames := map[string]string{}
	methods := []string{}
	for i := 0; i < rt.NumMethod(); i++ {
		m := rt.Method(i)
		methodNames[strings.ToLower(m.Name)] = m.Name
		methods = append(methods, m.Name)
	}

	if method == "help" {
		return strings.Join(methods, "\n")
	}

	rv := reflect.ValueOf(obj)
	m := rv.MethodByName(methodNames[method])
	if m.Kind() != reflect.Func {
		return callUnknownFunction
	}
	mT := m.Type()

	argsv := []reflect.Value{}
	if mT.IsVariadic() {
		i := 0
		for ; i < mT.NumIn()-1; i++ {
			argsv = append(argsv, reflect.ValueOf(args[i]))
		}
		argsv = append(argsv, reflect.ValueOf(args[i:]))
	} else {
		for i := range args {
			argsv = append(argsv, reflect.ValueOf(args[i]))
		}
	}
	if len(argsv) < mT.NumIn() {
		return callTooFewArguments
	} else if len(argsv) > mT.NumIn() {
		return callTooManyArguments
	}
	if mT.IsVariadic() {
		return m.CallSlice(argsv)[0].String()
	} else {
		return m.Call(argsv)[0].String()
	}
}
