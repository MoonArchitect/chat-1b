package main

import (
	"fmt"
	"log"

	"github.com/gorilla/websocket"
)

// a binary that is started, tasked with being a simulated user for the chat
// - needs to interact with the server in a realistic way
// - record metrics (requests out per second, failures, avg message size, etc.)
// - test for errors
//// - check for correct message order

func main() {
	conn, resp, err := websocket.DefaultDialer.Dial("ws://localhost:8080/chat", nil)
	if err != nil {
		log.Fatal("fatal", err)
	}

	fmt.Println(resp.Status)
	// fmt.Println(conn)

	msg := struct {
		Opcode string `json:"opcode"`
		Data   struct {
			UID string `json:"uid"`
		} `json:"data"`
	}{
		Opcode: "hahahahahahhaha",
		Data: struct {
			UID string `json:"uid"`
		}{UID: "test_uid"},
	}

	for i := 0; i < 3; i++ {
		fmt.Println("Sending msg ", i, " : ", msg)
		err = conn.WriteJSON(msg)
		if err != nil {
			fmt.Println(err)
		}
	}
	fmt.Println("Closing...")
	err = conn.Close()
	if err != nil {
		fmt.Println("close error: ", err)
	}
	fmt.Println("Done")
}
