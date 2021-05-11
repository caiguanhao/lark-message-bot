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

	ChatName string
	ChatId   string
	UserId   string
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

func (c Call) Help() string {
	return getMethods(c)
}

func (c MastersCall) Help() string {
	return getMethods(c)
}

func (c MastersCall) Create(name ChatName) string {
	if name == "" {
		return "name is needed to create a chat, specify like this:\ncreate(name)"
	}
	chatId, err := c.http.lark.api.CreateChat(string(name), c.resp.Event.UserOpenId)
	if err != nil {
		log.Error(err)
		return err.Error()
	}
	return fmt.Sprintf(`chat with name "%s" has been created, its id is %s`, name, chatId)
}

func (c MastersCall) Join(chatIds ...ChatId) string {
	if len(chatIds) == 0 || chatIds[0] == "" {
		return "chat id is needed to join a chat, specify like this:\njoin(chat-id...)"
	}
	ret := []string{}
	for _, chatId := range chatIds {
		err := c.http.lark.api.AddUsersToChat(string(chatId), []string{c.resp.Event.UserOpenId})
		if err != nil {
			log.Error(err)
			ret = append(ret, err.Error())
			continue
		}
		ret = append(ret, fmt.Sprintf("successfully joined chat: %s", chatId))
	}
	return strings.Join(ret, "\n")
}

func (c MastersCall) Destroy(chatIds ...ChatId) string {
	if len(chatIds) == 0 || chatIds[0] == "" {
		return "chat id is needed to destroy a chat, specify like this:\ndestroy(chat-id...)"
	}
	ret := []string{}
	for _, chatId := range chatIds {
		err := c.http.lark.api.DestroyChat(string(chatId))
		if err != nil {
			log.Error(err)
			ret = append(ret, err.Error())
			continue
		}
		ret = append(ret, fmt.Sprintf("successfully destroyed chat: %s", chatId))
	}
	return strings.Join(ret, "\n")
}

func (c MastersCall) Members(chatId ChatId) string {
	if chatId == "" {
		return "chat id is needed to list members of a chat, specify like this:\nmembers(chat-id)"
	}
	group, err := c.http.lark.api.GetChatInfo(string(chatId))
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

func (c MastersCall) Add(chatId ChatId, userIds ...UserId) string {
	if chatId == "" || len(userIds) == 0 {
		return "chat id is needed to add users to a chat, specify like this:\nadd(chat-id, user-id...)"
	}
	var userIdStrs []string
	for _, userId := range userIds {
		userIdStrs = append(userIdStrs, string(userId))
	}
	err := c.http.lark.api.AddUsersToChat(string(chatId), userIdStrs)
	if err != nil {
		log.Error(err)
		return err.Error()
	}
	return "successfully added users to chat"
}

func (c MastersCall) Remove(chatId ChatId, userIds ...UserId) string {
	if chatId == "" || len(userIds) == 0 {
		return "chat id is needed to remove users from a chat, specify like this:\nremove(chat-id, user-id...)"
	}
	var userIdStrs []string
	for _, userId := range userIds {
		userIdStrs = append(userIdStrs, string(userId))
	}
	err := c.http.lark.api.RemoveUsersFromChat(string(chatId), userIdStrs)
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
	for i := 0; i < rt.NumMethod(); i++ {
		m := rt.Method(i)
		methodNames[strings.ToLower(m.Name)] = m.Name
	}

	rv := reflect.ValueOf(obj)
	m := rv.MethodByName(methodNames[method])
	if m.Kind() != reflect.Func {
		return callUnknownFunction
	}
	mT := m.Type()

	if mT.IsVariadic() {
		if len(args) < mT.NumIn()-1 {
			return callTooFewArguments
		}

		argsv := []reflect.Value{}
		i := 0
		for ; i < mT.NumIn()-1; i++ {
			v := reflect.New(mT.In(i)).Elem()
			v.SetString(args[i])
			argsv = append(argsv, v)
		}
		vt := mT.In(i)
		argsv2 := reflect.New(vt).Elem()
		for ; i < len(args); i++ {
			v := reflect.New(vt.Elem()).Elem()
			v.SetString(args[i])
			argsv2 = reflect.Append(argsv2, v)
		}
		argsv = append(argsv, argsv2)
		return m.CallSlice(argsv)[0].String()
	}

	if len(args) < mT.NumIn() {
		return callTooFewArguments
	} else if len(args) > mT.NumIn() {
		return callTooManyArguments
	}

	argsv := []reflect.Value{}
	for i := range args {
		v := reflect.New(mT.In(i)).Elem()
		v.SetString(args[i])
		argsv = append(argsv, v)
	}
	return m.Call(argsv)[0].String()
}

func getMethods(c interface{}) string {
	var help []string
	rt := reflect.TypeOf(c)
	for i := 0; i < rt.NumMethod(); i++ {
		method := rt.Method(i)
		str := method.Name + "("
		mt := method.Type
		var args []string
		for j := 1; j < mt.NumIn(); j++ {
			name := mt.In(j).Name()
			if mt.In(j).Kind() == reflect.Slice {
				name = mt.In(j).Elem().Name()
				if mt.IsVariadic() && j == mt.NumIn()-1 {
					name += "..."
				} else {
					name += "[]"
				}
			}
			args = append(args, name)
		}
		str += strings.Join(args, ", ") + ")"
		help = append(help, str)
	}
	return strings.Join(help, "\n")
}
