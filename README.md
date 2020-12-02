# lark-message-bot

A simple message bot for Lark (Feishu / 飞书).

This is a small server application, you may need to configure nginx to serve
over https and control access.

Steps to create Feishu bot:

1. Create Custom App
2. Go to Features, enable "Using Bot"
3. Go to Event Subscriptions page, set Request URL to this server address.
   And add "Accept to messages" event.
4. Create a new version, set availability status to all, publish it and wait
   for review.
5. You may need to add more permissions for the `members` command.

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
