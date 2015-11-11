package main

import (
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/golang-lru"
	"github.com/microamp/slacko/config"
	"github.com/microamp/slacko/utils"
	"github.com/nlopes/slack"
)

const (
	configFile          = "slacko.json"
	envVarSlackAPIToken = "SLACK_API_TOKEN"
)

var cache *lru.Cache // LRU cache

// Filter out irrelevant request messages
func filterRequests(unfiltered <-chan *utils.MessageContext, filtered chan<- *utils.MessageContext) {
	for {
		mc, ok := <-unfiltered
		if !ok {
			mc.Printf("Error receiving from channel\n")
		}
		sInfo, err := mc.GetInfo()
		if err != nil {
			mc.Printf("Error retrieving message info: %v\n", err)
			continue
		}
		mc.Printf("Message before filter: %s\n", sInfo)

		// Ignore messages from bots (including @slacko itself)
		// unless it's edited (no user)
		if !mc.MsgEdited {
			isBot, err := mc.IsBot()
			if err != nil {
				mc.Printf("Error getting user info: %v\n", err)
				continue
			}
			if isBot {
				mc.Printf("Ignoring message from other bots\n")
				continue
			}
		}

		// Ignore messages unless starting with '@slacko: '
		replyToID := mc.ExtractReplyToID()
		if replyToID == "" {
			mc.Printf("Ignoring message without '@<user>: ' prefix\n")
			continue
		}
		replyToInfo, err := mc.GetUserInfo(replyToID)
		if err != nil {
			mc.Printf("Error getting user info: %v\n", err)
			continue
		}
		if replyToInfo.Name != mc.BotName {
			mc.Printf("%s is not %s\n", replyToInfo.Name, mc.BotName)
			continue
		}
		mc.ReplyToID = replyToID

		filtered <- mc
	}
}

// Build response messages
func buildResponses(filtered <-chan *utils.MessageContext, gpClient utils.GoPlaygroundClient) {
	for {
		mc, ok := <-filtered
		if !ok {
			mc.Printf("Error receiving from channel\n")
			continue
		}
		sInfo, err := mc.GetInfo()
		if err != nil {
			mc.Printf("Error retrieving message info: %v\n", err)
			continue
		}
		mc.Printf("Message after filter: %s\n", sInfo)

		params := slack.PostMessageParameters{
			Username: mc.BotName,
			AsUser:   true,
		}

		// Ignore all non-code messages
		code := mc.ExtractCode(mc.ReplyToID)
		if code == "" {
			_, ts, err := mc.PostMessage(
				mc.MsgChannel,
				fmt.Sprintf("Error: %s\n", "No code received. Accepted formats are\n"+
					"`single-line code`\n"+
					"```multi-line code```"),
				params,
			)
			if err != nil {
				mc.Printf("Error posting message: %v\n", err)
				continue
			}
			go cache.Add(mc.MsgTimestamp, ts)
			continue
		}

		// Compile code via Go Playground
		result, err := gpClient.Compile(code)

		// Post unexpected error
		if err != nil {
			_, ts, err := mc.PostMessage(
				mc.MsgChannel,
				fmt.Sprintf("Error compiling: %v\n", err),
				params,
			)
			if err != nil {
				mc.Printf("Error posting message: %v\n", err)
				continue
			}
			go cache.Add(mc.MsgTimestamp, ts)
			continue
		}

		// Post compile errors from Go Playground
		if result.CompileErrors != "" {
			_, ts, err := mc.PostMessage(
				mc.MsgChannel,
				fmt.Sprintf("Compile errors from Go Playground: %s", result.CompileErrors),
				params,
			)
			if err != nil {
				mc.Printf("Error posting message: %v\n", err)
				continue
			}
			go cache.Add(mc.MsgTimestamp, ts)
			continue
		}

		resultOutput := result.GetOutput()

		// Check if edited
		if mc.MsgEdited {
			// Check if cached
			if cache.Contains(mc.MsgTimestamp) {
				cached, ok := cache.Get(mc.MsgTimestamp)
				if !ok {
					mc.Printf("Error updating message: %v\n", err)
					continue
				}

				// Update previous result
				_, _, _, err = mc.UpdateMessage(
					mc.MsgChannel,
					cached.(string),
					resultOutput,
				)
				if err != nil {
					mc.Printf("Error updating message: %v\n", err)
					continue
				}
			}
		} else {
			// Post new result
			_, ts, err := mc.PostMessage(
				mc.MsgChannel,
				resultOutput,
				params,
			)
			if err != nil {
				mc.Printf("Error posting message: %v\n", err)
				continue
			}
			go cache.Add(mc.MsgTimestamp, ts)
		}
	}
}

func main() {
	// Validate environment variable, SLACK_API_TOKEN
	slackAPIToken := os.Getenv(envVarSlackAPIToken)
	if slackAPIToken == "" {
		panic("API token not provided (env var: SLACK_API_TOKEN)")
	}

	// Validate config settings
	config, err := config.ReadConfig(configFile)
	if err != nil {
		panic(fmt.Sprintf("Error reading config (filename: %s): %v", configFile, err))
	}
	if config.GoPlaygroundHost == "" {
		panic("'GoPlaygroundHost' must be provided. Check your config.")
	}
	if config.BotName == "" {
		panic("'BotName' must be provided. Check your config.")
	}

	// Create LRU cache
	cache, err = lru.New(config.CacheSize)
	if err != nil {
		panic(fmt.Sprintf("Cannot create LRU cache: %v", err))
	}

	api := slack.New(slackAPIToken)
	api.SetDebug(config.DebugOn)

	rtm := api.NewRTM()
	go rtm.ManageConnection()

	// Go Playground client
	gpClient := utils.GoPlaygroundClient{
		Host:    config.GoPlaygroundHost,
		DebugOn: config.DebugOn,
	}

	// Channel pipeline: unfiltered -> filtered
	unfiltered := make(chan *utils.MessageContext)
	filtered := make(chan *utils.MessageContext)

	go filterRequests(unfiltered, filtered)
	go buildResponses(filtered, gpClient)

Loop:
	for {
		select {
		case msg := <-rtm.IncomingEvents:
			switch event := msg.Data.(type) {
			case *slack.MessageEvent:
				mc := utils.NewMessageContext(api, event, config)
				go func() { unfiltered <- mc }()
			case *slack.RTMError:
				log.Printf("Error Receiving event: %s\n", event.Error())
			case *slack.InvalidAuthEvent:
				log.Printf("Invalid credentials")
				break Loop
			default:
				// Ignore all other types
			}
		}
	}
}
