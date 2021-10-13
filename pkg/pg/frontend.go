package pg

import (
	"crypto/md5"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	pgproto3 "github.com/jackc/pgproto3/v2"
)

const (
	// TODO: should this be configurable?
	readTimeout = 3 * time.Second
)

// Frontend acts as the postgres front-end client (ex: psql)
type Frontend interface {
	io.Closer
	Send(msg pgproto3.FrontendMessage) error
	SendRaw([]byte) error
	Receive() (pgproto3.BackendMessage, error)
	ReceiveRaw() ([]byte, error)
}

type AuthFailedError struct {
	ErrMsg *pgproto3.ErrorResponse
}

func (a *AuthFailedError) Error() string {
	return "auth failed"
}

// PostgresFrontend implements a postgres frontend client
type PostgresFrontend struct {
	frontend    *pgproto3.Frontend
	connection  net.Conn
	IdleTimeout time.Duration
	Mutex       sync.Mutex
}

// FrontendOption allows us to specify options
type FrontendOption func(f *PostgresFrontend) error

// NewFrontend returns a new postgres frontend
func NewFrontend(conn net.Conn, opts ...FrontendOption) (*PostgresFrontend, error) {
	f := &PostgresFrontend{
		frontend:    pgproto3.NewFrontend(pgproto3.NewChunkReader(conn), conn),
		connection:  conn,
		IdleTimeout: readTimeout,
		Mutex:       sync.Mutex{},
	}

	for _, opt := range opts {
		err := opt(f)
		if err != nil {
			return nil, err
		}
	}
	return f, nil
}

// Send sends a frontend message to the backend
func (f *PostgresFrontend) Send(msg pgproto3.FrontendMessage) error {
	f.Mutex.Lock()
	err := f.frontend.Send(msg)
	f.Mutex.Unlock()
	return err
}

// SendRaw sends arbitrary bytes to a backend
func (f *PostgresFrontend) SendRaw(b []byte) error {
	_, err := f.connection.Write(b)
	return err
}

// Receive accepts a message from the backend, or errors if nothing is read
// within the idle timeout. Returns io.ErrUnexpectedEOF if the connection has
// been closed.
func (f *PostgresFrontend) Receive() (pgproto3.BackendMessage, error) {
	_ = f.connection.SetReadDeadline(time.Now().Add(f.IdleTimeout))
	return f.frontend.Receive()
}

// ReceiveRaw accepts a message from the backend, or errors if nothing
// is read within the idle timeout.  Returns io.ErrUnexpectedEOF if the
// connection has been closed.
func (f *PostgresFrontend) ReceiveRaw() ([]byte, error) {
	// Postgres send buffers are at least this large
	response := make([]byte, 8192)
	_ = f.connection.SetReadDeadline(time.Now().Add(f.IdleTimeout))
	readBytes, err := f.connection.Read(response)
	if err != nil && err == io.EOF {
		err = io.ErrUnexpectedEOF
	}
	return response[:readBytes], err
}

// Close closes the underlying connection
func (b *PostgresFrontend) Close() error {
	return b.connection.Close()
}

func (f *PostgresFrontend) HandleAuthenticationRequest(username, password string) error {
	// TODO: max tries / exit condition?
	for {
		message, err := f.frontend.Receive()
		if err != nil {
			return err
		}

		switch msg := message.(type) {
		case *pgproto3.AuthenticationOk:
			return nil
		case *pgproto3.ReadyForQuery:
			return nil
		case *pgproto3.AuthenticationMD5Password:
			if err = f.Send(createMd5(msg, username, password)); err != nil {
				return err
			}
			continue
		case *pgproto3.AuthenticationCleartextPassword:
			if err := f.Send(createCleartext(msg, username, password)); err != nil {
				return err
			}
			continue
		case *pgproto3.ErrorResponse:
			return &AuthFailedError{ErrMsg: msg}
		case *pgproto3.AuthenticationSASL:
			return fmt.Errorf("SASL auth not supported")
		default:
			return fmt.Errorf("unsupported auth request, or unexpected message")
		}
	}
}

func createMD5Password(username string, password string, salt string) string {
	// Concatenate the password and the username together.
	passwordString := fmt.Sprintf("%s%s", password, username)

	// Compute the MD5 sum of the password+username string.
	passwordString = fmt.Sprintf("%x", md5.Sum([]byte(passwordString)))

	// Compute the MD5 sum of the password hash and the salt
	passwordString = fmt.Sprintf("%s%s", passwordString, salt)
	return fmt.Sprintf("md5%x", md5.Sum([]byte(passwordString)))
}

func createMd5(msg *pgproto3.AuthenticationMD5Password, username, password string) *pgproto3.PasswordMessage {
	return &pgproto3.PasswordMessage{
		Password: createMD5Password(username, password, string(msg.Salt[:])),
	}
}

func createCleartext(msg *pgproto3.AuthenticationCleartextPassword, username, password string) *pgproto3.PasswordMessage {
	return &pgproto3.PasswordMessage{
		Password: password,
	}
}
