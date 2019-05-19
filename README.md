slacko
======
Slack bot meets Go Playground

Demos
-----
* Single-line code

![](https://cdn.rawgit.com/microamp/slacko/develop/gifs/slacko1-new.gif)

* Multi-line code

![](https://cdn.rawgit.com/microamp/slacko/develop/gifs/slacko2-new.gif)

Dependencies
------------
* [goimports](https://github.com/bradfitz/goimports)
```
go get golang.org/x/tools/cmd/goimports
```
* [nlopes/slack](https://github.com/nlopes/slack)
```
go get github.com/nlopes/slack
```
* [hashicorp/golang-lru](https://github.com/hashicorp/golang-lru)
```
go get github.com/hashicorp/golang-lru
```

Configuration
-------------
Config settings are stored in a JSON file, `slacko.json`, consisting of the following keys:
* `GoPlaygroundHost`: Go Playground server (default: http://play.golang.org/compile?output=json)
* `BotName`: bot name (default: slacko)
* `DebugOn`: flag for enabling/disabling debug messages (default: true)
* `CacheSize`: LRU cache size (default: 128)

In addition, slacko expects the environment variable, `SLACK_API_TOKEN`, to store your bot's API token. You can set it by running the following command:
```
export SLACK_API_TOKEN="your_slack_bot_api_token"
```

License
-------
BSD 2-clause "Simplified" License
