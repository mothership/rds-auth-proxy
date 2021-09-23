package proxy

import (
	"context"
	"io"
	"net"
	"sync"

	"github.com/mothership/rds-auth-proxy/pkg/log"
	"go.uber.org/zap"
)

// errorWrapper wraps an error from a particular proxy
type errorWrapper struct {
	ConnectionID uint64
	Error        error
}

// Manager watches a group of proxies
type Manager struct {
	ActiveSessions sync.Map
	errorCh        chan errorWrapper
	cfg            *Config
}

// NewManager returns an instance of Manager
func NewManager(opts ...Option) (*Manager, error) {
	cfg := &Config{}
	for _, opt := range opts {
		err := opt(cfg)
		if err != nil {
			return nil, err
		}
	}
	return &Manager{
		ActiveSessions: sync.Map{},
		// Should probably have a similar buffer size to active sessions?
		errorCh: make(chan errorWrapper, 10),
		cfg:     cfg,
	}, nil
}

// Start starts the proxy server
func (m *Manager) Start(ctx context.Context) error {
	go m.errorHandler(ctx)
	listener, err := net.ListenTCP("tcp", m.cfg.ListenAddress)
	if err != nil {
		return err
	}

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			log.Error("error accepting connection from client", zap.Error(err))
			continue
		}
		log.Info(
			"accepted connection from client",
			zap.String("client_address", conn.RemoteAddr().String()),
		)
		p := newProxy(conn, m.errorCh, m.cfg)
		m.ActiveSessions.Store(p.ID, p)
		//nolint:errcheck // Errors are handled in m.errorCh
		go p.Start()
	}
}

func (m *Manager) errorHandler(ctx context.Context) {
	log.Info("starting error handler")
	defer log.Debug("shut down error handler")
	for {
		select {
		case <-ctx.Done():
			return
		case err := <-m.errorCh:
			if err.Error != nil && err.Error != io.ErrUnexpectedEOF {
				log.Error("proxy caught error",
					zap.Uint64("connectionID", err.ConnectionID),
					zap.Error(err.Error),
				)
			}
			if p, loaded := m.ActiveSessions.LoadAndDelete(err.ConnectionID); loaded {
				proxy, _ := p.(*Proxy)
				log.Info("stopping proxy", zap.Uint64("connectionID", err.ConnectionID))
				proxy.Stop()
				log.Info("proxy stopped", zap.Uint64("connectionID", err.ConnectionID))
			}
		}
	}

}
