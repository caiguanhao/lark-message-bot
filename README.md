# lark-message-bot

A simple message bot for Lark (Feishu / 飞书).

This is a small server application, you may need to configure nginx to serve over https.

Steps to create Feishu bot:

1. Create Custom App
2. Go to Permissions page, add "Read group information"
3. Go to Event Subscriptions page, set Request URL to this server address.
   And add "Accept to messages" event.
4. Create a new version, set availability status to all, publish it and wait for review.
