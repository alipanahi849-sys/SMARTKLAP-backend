// Command ws_probe connects to the realtime WebSocket, subscribes to a match
// channel, and prints every event received. Used for manual end-to-end testing
// of the chant.upcoming notification.
//
// Usage: go run ./scripts/ws_probe -token <jwt> -match <matchID> [-timeout 90s]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	token := flag.String("token", "", "JWT access token")
	match := flag.String("match", "", "match UUID to subscribe to (optional with -nosub)")
	nosub := flag.Bool("nosub", false, "connect without subscribing to any channel")
	addr := flag.String("addr", "ws://localhost:8080/api/v1/realtime/ws", "WebSocket URL")
	timeout := flag.Duration("timeout", 90*time.Second, "how long to wait for events")
	flag.Parse()

	if *token == "" || (*match == "" && !*nosub) {
		fmt.Fprintln(os.Stderr, "usage: ws_probe -token <jwt> [-match <matchID> | -nosub]")
		os.Exit(2)
	}

	header := http.Header{"Authorization": {"Bearer " + *token}}
	conn, _, err := websocket.DefaultDialer.Dial(*addr, header)
	if err != nil {
		fmt.Fprintln(os.Stderr, "dial failed:", err)
		os.Exit(1)
	}
	defer conn.Close()
	fmt.Println("connected to", *addr)

	if *nosub {
		fmt.Println("not subscribing to any channel (-nosub)")
	} else {
		sub := map[string]string{"type": "subscribe", "channel": "match:" + *match}
		if err := conn.WriteJSON(sub); err != nil {
			fmt.Fprintln(os.Stderr, "subscribe failed:", err)
			os.Exit(1)
		}
		fmt.Println("subscribed to match:" + *match)
	}

	deadline := time.Now().Add(*timeout)
	_ = conn.SetReadDeadline(deadline)

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			fmt.Fprintln(os.Stderr, "read ended:", err)
			os.Exit(1)
		}
		var pretty map[string]any
		if json.Unmarshal(raw, &pretty) == nil {
			out, _ := json.MarshalIndent(pretty, "", "  ")
			fmt.Printf("--- event @ %s ---\n%s\n", time.Now().Format("15:04:05.000"), out)
			if pretty["type"] == "chant.upcoming" {
				fmt.Println("SUCCESS: chant.upcoming received")
				return
			}
		} else {
			fmt.Println("raw:", string(raw))
		}
	}
}
