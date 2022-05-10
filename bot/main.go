package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/anonyindian/gotgproto"
	"github.com/anonyindian/gotgproto/dispatcher/handlers"
	"github.com/anonyindian/gotgproto/dispatcher/handlers/filters"
	"github.com/anonyindian/gotgproto/sessionMaker"
	"github.com/gotd/td/telegram"

	// "github.com/aws/aws-lambda-go/events"
	// "github.com/aws/aws-lambda-go/lambda"

	"github.com/anonyindian/gotgproto/dispatcher"
	"github.com/anonyindian/gotgproto/ext"
	"github.com/gotd/td/tg"
)

// Response is of type APIGatewayProxyResponse since we're leveraging the
// AWS Lambda Proxy Request functionality (default behavior)
//
// https://serverless.com/framework/docs/providers/aws/events/apigateway/#lambda-proxy-integration
// type Response events.APIGatewayProxyResponse

// // Handler is our lambda handler invoked by the `lambda.Start` function call
// func Handler(ctx context.Context) (Response, error) {
// 	var buf bytes.Buffer

// 	mydata := []byte("All the data I wish to write to a file")

// 	// the WriteFile method returns an error if unsuccessful
// 	err := ioutil.WriteFile("/tmp/myfile.session", mydata, 0777)
// 	// handle this error
// 	if err != nil {
// 		// print it out
// 		log.Panic(err)
// 	}

// 	data, err := ioutil.ReadFile("/tmp/myfile.session")
// 	if err != nil {
// 		fmt.Println(err)
// 	}

// 	body, err := json.Marshal(map[string]interface{}{
// 		"message": string(data),
// 	})
// 	if err != nil {
// 		return Response{StatusCode: 404}, err
// 	}
// 	json.HTMLEscape(&buf, body)

// 	response := Response{
// 		StatusCode:      200,
// 		IsBase64Encoded: false,
// 		Body:            buf.String(),
// 		Headers: map[string]string{
// 			"Content-Type":           "application/json",
// 			"X-MyCompany-Func-Reply": "world-handler",
// 		},
// 	}

// 	return response, nil
// }

var SERVICE_STARTED bool
var CHANGE_STATUS_SERVICE sync.Mutex

func changeStatusService(val bool) {

	CHANGE_STATUS_SERVICE.Lock()
	SERVICE_STARTED = val
	CHANGE_STATUS_SERVICE.Unlock()
}

func getStatusService() bool {

	CHANGE_STATUS_SERVICE.Lock()
	status := SERVICE_STARTED
	CHANGE_STATUS_SERVICE.Unlock()

	return status
}

func main() {

	changeStatusService(false)

	os.Setenv("BOT_TOKEN", "")
	os.Setenv("APP_ID", "")
	os.Setenv("API_HASH", "")

	appID, err := strconv.Atoi(os.Getenv("APP_ID"))

	if err != nil {
		log.Panic(err)
	}

	// custom dispatcher handles all the updates
	dp := dispatcher.MakeDispatcher()
	gotgproto.StartClient(gotgproto.ClientHelper{
		// Get AppID from https://my.telegram.org/apps
		AppID: appID,
		// Get ApiHash from https://my.telegram.org/apps
		ApiHash: os.Getenv("API_HASH"),
		// Session of your client
		// sessionName: name of the session / session string in case of TelethonSession or StringSession
		// sessionType: can be any out of Session, TelethonSession, StringSession.
		Session: sessionMaker.NewSession("tmp/info-contacts", sessionMaker.Session),
		// Get BotToken from @botfather
		BotToken: os.Getenv("BOT_TOKEN"),
		// Make sure to specify custom dispatcher here in order to enjoy gotgproto's update handling
		Dispatcher: dp,
		// Add the handlers, post functions in TaskFunc
		TaskFunc: func(ctx context.Context, client *telegram.Client) error {
			// Command Handler for /start
			dp.AddHandler(handlers.NewCommand("start", startMonitoringService))
			dp.AddHandler(handlers.NewCommand("stop", stopMonitoringService))
			// Callback Query Handler with prefix filter for recieving specific query
			dp.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Equal("initMonitoring"), initMonitoring))
			// This Message Handler will call our echo function on new messages
			dp.AddHandlerToGroup(handlers.NewMessage(filters.Message.Text, echo), 1)
			go func() {
				for {
					if gotgproto.Sender != nil {
						break
					}
				}
			}()
			return nil
		},
	})

	// lambda.Start(Handler)
}

// callback function for /start command
func startMonitoringService(ctx *ext.Context, update *ext.Update) error {
	user := update.EffectiveUser()
	ctx.Reply(update, fmt.Sprintf("Hello %s, I am @%s and will repeat all your messages", user.FirstName, ctx.Self.Username), &ext.ReplyOpts{
		Markup: &tg.ReplyInlineMarkup{
			Rows: []tg.KeyboardButtonRow{
				{
					Buttons: []tg.KeyboardButtonClass{
						&tg.KeyboardButtonCallback{
							Text: "Init",
							Data: []byte("initMonitoring"),
						},
					},
				},
			},
		},
	})
	// End dispatcher groups so that bot doesn't echo /start command usage
	return dispatcher.EndGroups
}

func stopMonitoringService(ctx *ext.Context, update *ext.Update) error {
	changeStatusService(false)
	return dispatcher.EndGroups
}

func initMonitoring(ctx *ext.Context, update *ext.Update) error {

	query := update.CallbackQuery
	changeStatusService(true)

	for getStatusService() {

		time.Sleep(1000 * time.Millisecond)
	}

	ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
		Alert:   true,
		QueryID: query.QueryID,
		Message: "service stoped",
	})

	return nil
}

func echo(ctx *ext.Context, update *ext.Update) error {
	msg := update.EffectiveMessage
	_, err := ctx.Reply(update, msg.Message, nil)
	return err
}
