# lark-message-bot

A simple message bot for Lark (Feishu / 飞书).

This is a small server application, you may need to configure nginx to serve over https.

Steps to create Feishu bot:

1. Create Custom App
2. Go to Features, enable "Using Bot"
3. Go to Event Subscriptions page, set Request URL to this server address.
   And add "Accept to messages" event.
4. Create a new version, set availability status to all, publish it and wait for review.

Once the bot is available, send `+` to the bot and it will create a new chat.
The Chat ID is stored in the `chat.id` file by default.
Others can send `+` to join the same chat to receive messages.

Make an HTTP JSON-RPC request to send message to the chat (so that everyone in the chat receives this message).

```
curl http://127.0.0.1:32123/jsonrpc/ --data '{"method":"Lark.SendMessage","params":[{"content":"test"}]}'
```
