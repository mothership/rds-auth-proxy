package examples

import (
	"fmt"
	"time"

	"github.com/jackc/pgproto3/v2"
	"github.com/mothership/rds-auth-proxy/pkg/pg"
	"github.com/mothership/rds-auth-proxy/pkg/proxy"
)

// BasicInterceptor just echoes back the query to the backend.
// Since it returns nil, the proxy will handle sending the message to the frontend.
func BasicInterceptor(frontend pg.SendOnlyFrontend, backend pg.SendOnlyBackend, msg *pgproto3.Query) error {
	message := fmt.Sprintf("Got query from client: %+v", msg.String)
	_ = backend.Send(&pgproto3.NoticeResponse{Message: message})
	return nil
}

// BasicDelayedInterceptor calls a goroutine and tells the proxy it
// will take care of sending the message to the frontend.
func BasicDelayedInterceptor(frontend pg.SendOnlyFrontend, backend pg.SendOnlyBackend, msg *pgproto3.Query) error {
	go func(frontend pg.SendOnlyFrontend, backend pg.SendOnlyBackend, msg *pgproto3.Query) {
		message := "Starting long running task. Please wait."
		_ = backend.Send(&pgproto3.NoticeResponse{Message: message})
		time.Sleep(time.Second * 5)
		_ = frontend.Send(msg)
	}(frontend, backend, msg)
	return fmt.Errorf(proxy.WillSendManually)
}
