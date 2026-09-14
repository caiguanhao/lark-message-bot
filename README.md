# lark-message-bot

A simple message bot for Lark (Feishu / 飞书).

This is a small server application. It receives Feishu events through an outbound
WebSocket long connection, so it does not need a public event callback URL.

## Options

```
app-id                 - lark app id (you can also use env LARK_APP_ID)
app-secret             - lark app secret (you can also use env LARK_APP_SECRET)
listen                 - private http server address
verbosity              - debug, info, notice, warning, error, critical
masters                - user ids to work for, separated by ","
aliyun-access-key-id   - aliyun access key id (you can also use env ALIYUN_ACCESS_KEY_ID)
aliyun-access-key-secret - aliyun access key secret (you can also use env ALIYUN_ACCESS_KEY_SECRET)
aliyun-main-domain     - main domain for create_subdomain command (you can also use env ALIYUN_MAIN_DOMAIN)
aliyun-default-ip      - default A record IP for create_subdomain command (you can also use env ALIYUN_DEFAULT_IP)
```

## Creation

Steps to create Feishu bot:

1. Create Custom App
2. Go to Features, enable "Using Bot"
3. Go to Event Subscriptions and select "Use long connection to receive events".
   Add "Receive message v2.0" (im.message.receive_v1) with these permissions:
   - im:message.p2p_msg:readonly (read direct messages sent to the bot)
   - im:message.group_at_msg:readonly (receive group messages that @ the bot)
   - im:message.group_msg (if you want to receive all group messages)
4. Keep the bot process running with outbound access to Feishu. No Request URL,
   Verification Token, Encrypt Key, or nginx event route is required.
5. Create a new version, set availability status to all, publish it and wait
   for review.
6. You may need to add more permissions for the `members` command.

## Commands

Available commands that you can send to the bot:

### create(name)

Create new chat with name and add you to the new chat.

### join(chat_id...)

Add you to one or more chats.

### destroy(chat_id...)

Destroy one or more chats.

### members(chat_id)

List names and user ids of the chat with chat_id.

Extra permissions for this command:
- Access user basic information from the contact
- Access contacts by application identity

### add(chat_id, user_id...)

Add one or more users to chat.

### remove(chat_id, user_id...)

Remove one or more users from chat.

### list

List all chats created by the bot.

### create_subdomain(sub_domain, ip_address...)

Create a new subdomain `<sub_domain>.<aliyun-main-domain>` with an A record
pointing to `aliyun-default-ip`, using Aliyun DNS. For example, if the main
domain is `example.com`:

```
create_subdomain(mdycgl.fs)
```

creates `mdycgl.fs.example.com`. You can also pass a full domain name like
`mdycgl.fs.example.com`, the main domain suffix is stripped automatically.
Command names are case and underscore insensitive, so `CREATE_SUBDOMAIN(...)`
also works. If you copy and paste a command in lark, pasted domains may turn
into links (`[domain](http://domain/)` or `<a href="...">domain</a>`); they are
unwrapped automatically before parsing.

To use a custom IP address instead of the default one, pass it as the second
argument:

```
create_subdomain(abc, 127.0.0.1)
```

The aliyun access key needs the permission to manage DNS records
of the main domain (e.g. `AliyunDNSFullAccess`).

### list_subdomains

List all A records of `aliyun-main-domain`, newest first:

```
list_subdomains
```

outputs lines like:

```
mdycgl.fs.example.com -> 127.0.0.1 (created: 2026-09-14 10:00:00)
```

Only the newest 50 records are shown if there are more.

### whosyourdaddy

Show the author of the bot.

### whoami

Show your user id.

## JSON-RPC

Make an HTTP JSON-RPC request to send message to the chat (so that everyone in
the chat receives this message).

```
curl http://127.0.0.1:32123/jsonrpc/ --data \
  '{"method":"Lark.SendMessage","params":[{"chat_id":"oc_xxxx","content":"test"}]}'
```

### Lark.SendMessage

Send plain text to chat.

```js
{
  "chat_id": "oc_xxxx",
  "content": "text"
}
```

### Lark.SendPost

Send message with text, links, images to chat.
See [docs](https://open.feishu.cn/document/ukTMukTMukTM/uMDMxEjLzATMx4yMwETM).

```js
{
  "chat_id": "oc_xxxx",
  "post": {
    "zh_cn": {
      "title": "Title",
      "content": [
        [
          {
            "tag": "text",
            "text": "Test: "
          },
          {
            "tag": "a",
            "text": "Google",
            "href": "https://www.google.com"
          }
        ],
        [
          {
            "tag": "text",
            "text": "some text"
          }
        ]
      ]
    }
  }
}
```

## nginx

nginx is only needed if you expose the local JSON-RPC endpoint. The bot does not
serve `/events/` or `/204/`; Feishu events use the outbound WebSocket connection.

Example:

```nginx
server {
	listen 443 ssl http2;
	server_name lark.yourdomain.com;
	location /jsonrpc/ {
		satisfy any;
		allow 8.9.10.11;   # ip addresses of your other servers
		allow 12.13.14.15; # ip addresses of your other servers
		deny all;
		auth_basic "closed site";
		auth_basic_user_file /etc/nginx/lark.passwd;
		proxy_pass http://127.0.0.1:32123;
	}
	location / {
		return 404;
	}
}
```
