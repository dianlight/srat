package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type WebSocketMessageEnvelope struct {
	Event string `json:"event"`
	Uid   string `json:"uid"`
	Data  any    `json:"data"`
	// Path      string `json:"path"`      // The name of the function. Can be a path or a method
	// Method    string `json:"method"`    // Method if the path is a REST method
	// Body      any    `json:"body"`      // The request body
	// Subcriber string `json:"subcriber"` // The UID of the subcriber
}

/*
type newresponsewriter struct {
	w      io.WriteCloser
	buf    bytes.Buffer
	code   int
	header http.Header
}

func (rw *newresponsewriter) Header() http.Header {
	return rw.header
}

func (rw *newresponsewriter) WriteHeader(statusCode int) {
	rw.code = statusCode
}

func (rw *newresponsewriter) Write(data []byte) (int, error) {
	return rw.buf.Write(data)
}

func (rw *newresponsewriter) Done() (int64, error) {
	//	if rw.code > 0 {
	//		rw.w.WriteHeader(rw.code)
	//	}
	return io.Copy(rw.w, &rw.buf)
}
*/

var upgrader = websocket.Upgrader{} // use default options

// WSChannel godoc
//
//	@Summary		WSChannel
//	@Description	Open the WSChannel
//	@Tags			system
//	@Produce		json
//	@Success		200
//	@Failure		405	{object}	ResponseError
//	@Router			/ws [get]
func WSChannelHandler(w http.ResponseWriter, rq *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	c, err := upgrader.Upgrade(w, rq, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	outchan := make(chan *WebSocketMessageEnvelope, 10)

	go func() {
		for {
			outmessage := <-outchan
			jsonResponse, jsonError := json.Marshal(&outmessage)

			if jsonError != nil {
				log.Printf("Unable to encode JSON")
				continue
			}
			if outmessage.Event != "heartbeat" {
				log.Printf("send: %s %s", outmessage.Event, string(jsonResponse))
			}
			c.WriteMessage(websocket.TextMessage, []byte(jsonResponse))
		}
	}()

	for {
		var message WebSocketMessageEnvelope
		err := c.ReadJSON(&message)
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("recv: %s %s", message.Event, message)
		// Dispatcher

		switch message.Event {
		case "heartbeat":
			go HealthCheckWsHandler(message, outchan)
		case "shares":
			go SharesWsHandler(message, outchan)
		case "volumes":
			go VolumesWsHandler(message, outchan)
		default:
			log.Printf("Unknown event: %s", message.Event)
		}

		/*
			var found mux.RouteMatch

				jsonBody, jsonError := json.Marshal(message.Body)

				if jsonError != nil {
					fmt.Println("Unable to encode JSON")
					continue
				}
		*/

		/*
			bogus_request := &http.Request{
				Method: http.MethodGet,
				URL: &url.URL{
					Path: message.Event,
				},
				//	Body: io.NopCloser(strings.NewReader(string(jsonBody))),
			}

			log.Printf("Bogus request: %s", bogus_request.URL.Path)
			log.Printf("Router: %s", globalRouter)
			if globalRouter.Match(bogus_request, &found) {
				writer, _ := c.NextWriter(websocket.TextMessage)
				httpWriter := &newresponsewriter{
					w:      writer,
					header: http.Header{},
				}
				found.Route.GetHandler().ServeHTTP(
					httpWriter,
					bogus_request,
				)
			}
		*/

		/*
			err = c.WriteMessage(mt, message)
			if err != nil {
				log.Println("write:", err)
				break
			}
		*/
	}
}
