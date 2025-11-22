package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/ManadaHerath/realtime-grid-server/internal/grid"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (api *API) HandleGridWS(w http.ResponseWriter, r *http.Request, gridID string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if api.Redis == nil {
		http.Error(w, "real-time not configured", http.StatusInternalServerError)
		return
	}

	if _, err := api.Store.GetGrid(gridID); err != nil {
		if err == grid.ErrGridNotFound {
			http.Error(w, "grid not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	channel := "grid:" + gridID + ":events"
	sub := api.Redis.Subscribe(ctx, channel)
	defer sub.Close()

	ch := sub.Channel()
	_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"hello","gridId":"`+gridID+`"}`))

	go func() {
		defer cancel()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload)); err != nil {
				fmt.Println("ws write error:", err)
				return
			}
		}
	}
}
