// Command ws_loadtest opens many concurrent WebSocket connections to measure
// how many clients the realtime stack can sustain before failures spike.
//
// Usage:
//
//	go run ./scripts/ws_loadtest \
//	  -token "$CLAP_WS_TOKEN" \
//	  -addr ws://localhost:8081/api/v1/realtime/ws \
//	  -clients 5000 -ramp 60s -duration 120s
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

type stats struct {
	attempted      atomic.Int64
	connected      atomic.Int64
	failedDial     atomic.Int64
	failedAuth     atomic.Int64
	closedEarly    atomic.Int64
	messagesRecv   atomic.Int64
	pingsSent      atomic.Int64
	subscribeFail  atomic.Int64
}

func main() {
	token := flag.String("token", os.Getenv("CLAP_WS_TOKEN"), "JWT access token (or CLAP_WS_TOKEN)")
	addr := flag.String("addr", "ws://localhost:8081/api/v1/realtime/ws", "WebSocket URL")
	clients := flag.Int("clients", 100, "target concurrent connections")
	ramp := flag.Duration("ramp", 30*time.Second, "ramp-up duration (0 = connect all at once)")
	duration := flag.Duration("duration", 60*time.Second, "hold duration after ramp (0 = until Ctrl+C)")
	match := flag.String("match", "", "optional match UUID to subscribe (match:<uuid>)")
	pingInterval := flag.Duration("ping", 5*time.Second, "app-level ping interval (0 = disable)")
	reportEvery := flag.Duration("report", 5*time.Second, "progress report interval")
	flag.Parse()

	if strings.TrimSpace(*token) == "" {
		fmt.Fprintln(os.Stderr, "error: -token or CLAP_WS_TOKEN is required")
		flag.Usage()
		os.Exit(2)
	}
	if *clients <= 0 {
		fmt.Fprintln(os.Stderr, "error: -clients must be > 0")
		os.Exit(2)
	}

	var st stats
	stop := make(chan struct{})
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		close(stop)
	}()

	header := http.Header{"Authorization": []string{"Bearer " + strings.TrimSpace(*token)}}
	channel := ""
	if *match != "" {
		channel = "match:" + strings.TrimSpace(*match)
	}

	fmt.Printf("ws_loadtest: addr=%s clients=%d ramp=%s duration=%s\n", *addr, *clients, *ramp, *duration)

	var wg sync.WaitGroup
	start := time.Now()
	rampDelay := time.Duration(0)
	if *ramp > 0 && *clients > 1 {
		rampDelay = *ramp / time.Duration(*clients-1)
	}

	for i := 0; i < *clients; i++ {
		if rampDelay > 0 && i > 0 {
			select {
			case <-stop:
				goto rampDone
			case <-time.After(rampDelay):
			}
		}

		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			runClient(id, *addr, header, channel, *pingInterval, &st, stop)
		}(i)

		select {
		case <-stop:
			goto rampDone
		default:
		}
	}
rampDone:

	reportTicker := time.NewTicker(*reportEvery)
	defer reportTicker.Stop()
	go func() {
		for {
			select {
			case <-stop:
				return
			case <-reportTicker.C:
				printProgress(&st, time.Since(start))
			}
		}
	}()

	if *duration > 0 {
		select {
		case <-stop:
		case <-time.After(*duration):
			close(stop)
		}
	} else {
		<-stop
	}

	wg.Wait()
	printSummary(&st, time.Since(start))
}

func runClient(id int, addr string, header http.Header, channel string, pingEvery time.Duration, st *stats, stop <-chan struct{}) {
	st.attempted.Add(1)

	dialer := websocket.Dialer{
		HandshakeTimeout: 15 * time.Second,
	}
	conn, resp, err := dialer.Dial(addr, header)
	if err != nil {
		st.failedDial.Add(1)
		if resp != nil && (resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden) {
			st.failedAuth.Add(1)
		}
		return
	}
	defer conn.Close()

	st.connected.Add(1)

	if channel != "" {
		sub := map[string]string{"type": "subscribe", "channel": channel}
		if err := conn.WriteJSON(sub); err != nil {
			st.subscribeFail.Add(1)
		}
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				return
			}
			st.messagesRecv.Add(1)
			var msg map[string]any
			if json.Unmarshal(raw, &msg) == nil {
				if t, _ := msg["type"].(string); t == "error" {
					st.subscribeFail.Add(1)
				}
			}
		}
	}()

	var pingTicker *time.Ticker
	var pingC <-chan time.Time
	if pingEvery > 0 {
		pingTicker = time.NewTicker(pingEvery)
		pingC = pingTicker.C
		defer pingTicker.Stop()
	}

	for {
		select {
		case <-stop:
			_ = conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, "load test done"))
			<-done
			return
		case <-done:
			st.closedEarly.Add(1)
			return
		case <-pingC:
			st.pingsSent.Add(1)
			ping := map[string]any{
				"type":           "ping",
				"client_time_ms": time.Now().UnixMilli(),
			}
			if err := conn.WriteJSON(ping); err != nil {
				st.closedEarly.Add(1)
				return
			}
		}
	}
}

func printProgress(st *stats, elapsed time.Duration) {
	attempted := st.attempted.Load()
	connected := st.connected.Load()
	failed := st.failedDial.Load()
	crashRate := pct(failed, attempted)
	fmt.Printf("[%s] connected=%d failed=%d crash_rate=%.1f%% msgs=%d pings=%d\n",
		elapsed.Round(time.Second),
		connected, failed, crashRate, st.messagesRecv.Load(), st.pingsSent.Load())
}

func printSummary(st *stats, elapsed time.Duration) {
	attempted := st.attempted.Load()
	connected := st.connected.Load()
	failed := st.failedDial.Load()
	fmt.Println()
	fmt.Println("========== ws_loadtest summary ==========")
	fmt.Printf("elapsed:           %s\n", elapsed.Round(time.Millisecond))
	fmt.Printf("attempted:         %d\n", attempted)
	fmt.Printf("connected (peak):  %d\n", connected)
	fmt.Printf("failed dial:       %d (%.1f%%)\n", failed, pct(failed, attempted))
	fmt.Printf("auth failures:     %d\n", st.failedAuth.Load())
	fmt.Printf("closed early:      %d\n", st.closedEarly.Load())
	fmt.Printf("subscribe errors:  %d\n", st.subscribeFail.Load())
	fmt.Printf("messages received: %d\n", st.messagesRecv.Load())
	fmt.Printf("pings sent:        %d\n", st.pingsSent.Load())
	fmt.Println("=========================================")

	if failed > 0 && float64(failed)/float64(max64(attempted, 1)) > 0.05 {
		os.Exit(1)
	}
}

func pct(part, total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(part) / float64(total) * 100
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
