package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// Global variables
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins
	},
}
var clients = make(map[*websocket.Conn]string) // Map connection to order_id

// WebSocket endpoint
func handleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}
	defer conn.Close()

	orderID := r.URL.Query().Get("order_id")
	clients[conn] = orderID

	// In this example, we don't expect to receive messages from the client,
	// but the loop is here to keep the connection open and handle any potential incoming messages.
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Printf("error: %v", err)
			delete(clients, conn)
			break
		}
	}
}

// Supabase webhook handler
func handleWebhook(w http.ResponseWriter, r *http.Request) {
	var data struct {
		OrderID string `json:"order_id"`
		Status  string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Send update to relevant WebSocket clients
	message := fmt.Sprintf(`{"order_id":"%s","status":"%s"}`, data.OrderID, data.Status)
	for conn, orderID := range clients {
		if orderID == data.OrderID {
			if err := conn.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
				log.Println("write:", err)
				delete(clients, conn)
			}
		}
	}
}

func main() {
	http.HandleFunc("/ws", handleConnections)
	http.HandleFunc("/webhook", handleWebhook)

	log.Println("Server started on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
