package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"html"
	"net"
	"reflect"
	"regexp"
	"strings"
	"time"
)

type (
	Call struct {
		senderOpenId string
		http         *httpHandler
	}

	MastersCall struct {
		Call
	}

	ChatName  string
	ChatId    string
	UserId    string
	SubDomain string
	IPAddress string
)

func (c Call) WhoAmI() string {
	return c.senderOpenId
}

func (c Call) WhosYourDaddy() string {
	var out []string
	for _, m := range c.http.mastersList {
		info, err := c.http.lark.api.GetUserInfo(m)
		if err == nil {
			out = append(out, fmt.Sprintf("%s (%s)", info.Name, info.OpenId))
		} else {
			log.Error(err)
			out = append(out, err.Error())
		}
	}
	return strings.Join(out, "\n")
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
	chatId, err := c.http.lark.api.CreateChat(string(name), c.senderOpenId)
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
		err := c.http.lark.api.AddUsersToChat(string(chatId), []string{c.senderOpenId})
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

func (c MastersCall) ChangeOwner(chatId ChatId, ownerId UserId) string {
	if chatId == "" || ownerId == "" {
		return "chat id or owner id is needed to change owner of a chat, specify like this:\nchangeowner(chat-id, user-id)"
	}
	err := c.http.lark.api.UpdateChat(string(chatId), map[string]interface{}{
		"owner_open_id": string(ownerId),
	})
	if err != nil {
		log.Error(err)
		return err.Error()
	}
	return "successfully changed owner of chat"
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
	var out []string
	for _, user := range userIds {
		info, err := c.http.lark.api.GetUserInfo(user)
		if err == nil {
			out = append(out, fmt.Sprintf("%s (%s)", info.Name, info.OpenId))
		} else {
			log.Error(err)
			out = append(out, err.Error())
		}
	}
	return strings.Join(out, "\n")
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

var subdomainRegexp = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)*$`)

func (c MastersCall) CreateSubdomain(subdomain SubDomain, ips ...IPAddress) string {
	if subdomain == "" {
		return "subdomain is needed to create a subdomain, specify like this:\ncreate_subdomain(sub-domain, ip-address...)"
	}
	ali := c.http.aliyun
	if !ali.configured() {
		return "aliyun dns is not configured, " +
			"please specify aliyun access key id, secret, main domain and default ip"
	}
	name := string(subdomain)
	// also accept a full domain name like sub.main-domain.com
	name = strings.TrimSuffix(name, "."+ali.MainDomain)
	if !subdomainRegexp.MatchString(name) {
		return fmt.Sprintf(`"%s" is not a valid subdomain name`, name)
	}
	ip := ali.DefaultIP
	if len(ips) > 0 && ips[0] != "" {
		ip = string(ips[0])
	}
	if parsed := net.ParseIP(ip); parsed == nil || parsed.To4() == nil {
		return fmt.Sprintf(`"%s" is not a valid IPv4 address`, ip)
	}
	recordId, err := ali.AddARecord(name, ip)
	if err != nil {
		log.Error(err)
		return err.Error()
	}
	return fmt.Sprintf(`subdomain "%s" has been created, it now resolves to %s (record id: %s)`,
		name+"."+ali.MainDomain, ip, recordId)
}

// maxSubdomainsShown limits the number of records listed by ListSubdomains
// to keep the chat message reasonably short.
const maxSubdomainsShown = 50

func (c MastersCall) ListSubdomains() string {
	ali := c.http.aliyun
	if !ali.configured() {
		return "aliyun dns is not configured, " +
			"please specify aliyun access key id, secret, main domain and default ip"
	}
	records, err := ali.ListARecords()
	if err != nil {
		log.Error(err)
		return err.Error()
	}
	if len(records) == 0 {
		return fmt.Sprintf(`no subdomains of "%s" found`, ali.MainDomain)
	}
	out := make([]string, 0, len(records)+1)
	for i, r := range records {
		if i >= maxSubdomainsShown {
			out = append(out, fmt.Sprintf("... and %d more", len(records)-maxSubdomainsShown))
			break
		}
		out = append(out, fmt.Sprintf("%s.%s -> %s (created: %s)",
			r.RR, ali.MainDomain, r.Value,
			time.Unix(r.CreateTimestamp/1000, 0).Format("2006-01-02 15:04:05")))
	}
	return strings.Join(out, "\n")
}

const (
	callUnknownExpression = "unknown expression"
	callUnknownFunction   = "unknown function"
	callTooFewArguments   = "too few arguments"
	callTooManyArguments  = "too many arguments"
)

func call(obj interface{}, expression string) string {
	expression = sanitizeExpression(expression)
	expr, err := parser.ParseExpr(maskIPAddresses(expression))
	if err != nil {
		log.Noticef("cannot parse expression %q: %s", expression, err)
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
	method = normalizeMethodName(method)

	rt := reflect.TypeOf(obj)
	methodNames := map[string]string{}
	for i := 0; i < rt.NumMethod(); i++ {
		m := rt.Method(i)
		methodNames[normalizeMethodName(m.Name)] = m.Name
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

var (
	markdownLinkRegexp = regexp.MustCompile(`\[([^]]*)\]\([^)]*\)`)
	htmlLinkRegexp     = regexp.MustCompile(`<a\b[^>]*>([^<]*)</a>`)
	bareURLRegexp      = regexp.MustCompile(`https?://([a-zA-Z0-9-]+(\.[a-zA-Z0-9-]+)+)/?`)
)

// sanitizeExpression unwraps markdown/html links, strips url schemes and
// unescapes html entities in the expression: lark turns pasted domains into
// markdown links, so CreateSubdomain(test.wanico.net) copied from a chat may
// arrive as CreateSubdomain([test.wanico.net](http://test.wanico.net/)).
func sanitizeExpression(expression string) string {
	expression = markdownLinkRegexp.ReplaceAllString(expression, "$1")
	expression = htmlLinkRegexp.ReplaceAllString(expression, "$1")
	expression = bareURLRegexp.ReplaceAllString(expression, "$1")
	return html.UnescapeString(expression)
}

var ipv4AddrRegexp = regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`)

// maskIPAddresses replaces dots in IPv4-address-like tokens (e.g. 127.0.0.1)
// with underscores so that the expression parses as Go syntax (127_0_0_1 is a
// valid integer literal). The replacement preserves byte offsets, so arguments
// can still be extracted from the original expression by position.
func maskIPAddresses(expression string) string {
	masked := []byte(expression)
	for _, loc := range ipv4AddrRegexp.FindAllStringIndex(expression, -1) {
		// skip matches that are part of a larger token like room1.2.3.4 or 1.2.3.4.5
		if loc[0] > 0 && isIPAddrTokenChar(expression[loc[0]-1]) {
			continue
		}
		if loc[1] < len(expression) && isIPAddrTokenChar(expression[loc[1]]) {
			continue
		}
		for i := loc[0]; i < loc[1]; i++ {
			if masked[i] == '.' {
				masked[i] = '_'
			}
		}
	}
	return string(masked)
}

func isIPAddrTokenChar(c byte) bool {
	return c == '.' || c == '_' ||
		(c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// normalizeMethodName makes method names case and underscore insensitive,
// so both createsubdomain(...) and create_subdomain(...) call CreateSubdomain.
func normalizeMethodName(name string) string {
	return strings.ReplaceAll(strings.ToLower(name), "_", "")
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
	help = append(help, "For more info, visit github.com/caiguanhao/lark-message-bot")
	return strings.Join(help, "\n")
}
