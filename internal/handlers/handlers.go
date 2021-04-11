package handlers

import (
	"fmt"
	"log"
	"net/http"
	"sort"

	"github.com/CloudyKit/jet/v6"
	"github.com/gorilla/websocket"
)

// wsChan is a channel that will only accept data of type WsPayload
var WsChan = make(chan WsPayload)

// clients is a map with a key of WebSocketConnection and a value of string
var clients = make(map[WebSocketConnection]string)

// loads the html template
// NewSet returns a new Set relying on a loader
var views = jet.NewSet(
	jet.NewOSFileSystemLoader("./html"),
	jet.InDevelopmentMode(), // take out in production
)

// upradeConnection used to upgrade connections to a websocket
var upgradeConnection = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// Home will be used to display a page
func Home(w http.ResponseWriter, r *http.Request) {
	// Every web handler in Go needs to have two arguments (w http.ResponseWriter and a pointer to a request r *http.Request)
	err := renderPage(w, "home.html", nil)
	if err != nil {
		log.Println(err)
	}

}

// wrapper for the websocket connection
type WebSocketConnection struct {
	*websocket.Conn
}

// WsJsonResponse defines the response sent back from websocket
type WsJsonResponse struct {
	Action         string   `json:"action"`
	Message        string   `json:"message"`
	MessageType    string   `json:"message_type"`
	ConnectedUsers []string `json:"connected_users"`
}

// WsPayload
type WsPayload struct {
	Action   string              `json:"action"`
	Username string              `json:"username"`
	Message  string              `json:"message"`
	Conn     WebSocketConnection `json:"-"`
}

// WsEndpoint upgrades connection to websocket
func WsEndpoint(w http.ResponseWriter, r *http.Request) {
	ws, err := upgradeConnection.Upgrade(w, r, nil)
	// declare variable ws, and err and upgrade to websocket protocol. args w, r, nil for reponse header
	if err != nil {
		log.Println(err)
	}
	log.Println("Client connected to endpoint") // confirm when someone connects to the homepage

	var response WsJsonResponse
	response.Message = `<em><small>Connected to server</small></em>`

	conn := WebSocketConnection{Conn: ws}
	clients[conn] = ""

	err = ws.WriteJSON(response) // err = pointer to websocket connection. WriteJSON writes the JSON encoding of v as a message.
	if err != nil {
		log.Println(err)
	}

	go WsListen(&conn) // when someone connects to the endpoint, the go routine will start
}

// Ws Listen listens for the web socket connection
func WsListen(conn *WebSocketConnection) {
	defer func() { // if an error occurs, recover and log the error.
		if r := recover(); r != nil {
			log.Printf("Error: %v", r)
		}
	}()

	var payload WsPayload // create a variable payload of type WsPayload struct

	for {
		err := conn.ReadJSON(&payload) // reads the json-encoded message from the payload
		if err != nil {
			// do nothing
		} else {
			payload.Conn = *conn
			WsChan <- payload // send data from the payload to the channel
		}
	}
}

// ListenWsChannel will listen for any data coming though on WsChan, then respond
func ListenWsChannel() {
	var response WsJsonResponse
	for {
		e := <-WsChan

		switch e.Action {
		case "username":
			// get a list of all users and send it back via broadcast
			clients[e.Conn] = e.Username
			users := getUserList()
			response.Action = "list_users"
			response.ConnectedUsers = users
			broadcaster(response)

			// delete users as they leave
		case "left":
			response.Action = "list_users"
			delete(clients, e.Conn)
			users := getUserList()
			response.ConnectedUsers = users
			broadcaster(response)

			// prints name and message in chat
		case "broadcast":
			response.Action = "broadcast"
			response.Message = fmt.Sprintf("<strong>%s</strong>: %s", e.Username, e.Message)
			broadcaster(response)
		}
	}
}

func getUserList() []string {
	var userList []string
	for _, x := range clients {
		if x != "" {
			userList = append(userList, x)
		}
	}
	sort.Strings(userList)
	return userList
}

func broadcaster(response WsJsonResponse) {
	for client := range clients { // for every client listed, send the json-encoded response
		err := client.WriteJSON(response)
		if err != nil { // if client doesn't exist or leaves, log err, close connection, and remove client
			log.Println("websocket err")
			_ = client.Close()
			delete(clients, client)
		}
	}
}

// renderPage will be used for any handler that needs to render a page,
// takes 3 args, http.ResponseWriter, template to render tmpl of type string, data passed to the template using jet.VarMap
func renderPage(w http.ResponseWriter, tmpl string, data jet.VarMap) error {
	view, err := views.GetTemplate(tmpl) // declaring a variable 'view' = to views.GetTemplate(tmpl) to render the template
	if err != nil {
		log.Println(err)
		return err
	}

	err = view.Execute(w, data, nil) // if no err occured above then execute data
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}