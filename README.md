# Telegram Serverless API

This is a serverless Go function initially intended to be deployed on Vercel.

I created it as an easier interface to public parts of Telegram (currently, public channels' messages).

Backlog:
- improve the API interface, by implementing:
  1. orderBy
  2. limit
  3. offset
- implement better control to guarantee Telegram's rate-limiting 
