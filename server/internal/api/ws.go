package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/dnahilman/scrapper-go/internal/logger"
	"github.com/dnahilman/scrapper-go/internal/logstream"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Accept connections from the same origin we serve from; permissive in dev.
	CheckOrigin: func(r *http.Request) bool { return true },
}

const (
	wsPongWait     = 60 * time.Second
	wsPingInterval = 25 * time.Second
	wsWriteWait    = 10 * time.Second
)

// registerWSRoute mounts GET /ws which streams logstream.Hub events to every
// connected client.
func registerWSRoute(r *gin.Engine, hub *logstream.Hub) {
	r.GET("/ws", func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			logger.L().Warn().Err(err).Msg("ws upgrade failed")
			return
		}
		serveWS(conn, hub)
	})
}

func serveWS(conn *websocket.Conn, hub *logstream.Hub) {
	log := logger.L().With().Str("svc", "ws").Logger()
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(wsPongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(wsPongWait))
		return nil
	})

	ch := hub.Subscribe()
	defer hub.Unsubscribe(ch)

	// Reader goroutine drains incoming frames (we don't expect any but must
	// service the read side for pongs / close frames to be processed).
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.NextReader(); err != nil {
				return
			}
		}
	}()

	ticker := time.NewTicker(wsPingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case ev, ok := <-ch:
			if !ok {
				_ = conn.WriteMessage(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseGoingAway, "subscription dropped"))
				return
			}
			conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if err := conn.WriteJSON(ev); err != nil {
				log.Debug().Err(err).Msg("write failed; closing client")
				return
			}
		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
