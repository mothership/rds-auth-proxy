package proxy_test

import (
	"fmt"
	"strings"
	"testing"

	. "github.com/mothership/rds-auth-proxy/pkg/proxy"
)

func TestConfigOptionErrors(t *testing.T) {
	config := Config{}

	cases := []struct {
		Option Option
		Error  error
	}{
		// Valid listen address
		{
			Option: WithListenAddress("0.0.0.0:8000"),
			Error:  nil,
		},
		// Missing port
		{
			Option: WithListenAddress("bah"),
			Error:  fmt.Errorf("missing port in address"),
		},
		// Bad Host
		{
			Option: WithListenAddress("bah:80"),
			// XXX: On OSX (local dev env) we get a different error message than testing on linux (CI)
			//      for now, just assert that we got an error :/
			Error: fmt.Errorf(""),
		},
		// Bad Port
		{
			Option: WithListenAddress("0.0.0.0:bah"),
			// XXX: On OSX (local dev env) we get a different error message than testing on linux (CI)
			//      for now, just assert that we got an error :/
			Error: fmt.Errorf(""),
		},
		// valid credential getter
		{
			Option: WithCredentialInterceptor(func(creds *Credentials) error {
				return nil
			}),
			Error: nil,
		},
		// valid mode
		{
			Option: WithMode(ServerSide),
			Error:  nil,
		},
		// invalid mode
		{
			Option: WithMode(Mode(10)),
			Error:  fmt.Errorf("invalid mode"),
		},
	}

	for idx, test := range cases {
		err := test.Option(&config)
		if !errorContains(err, test.Error) {
			t.Errorf("[Case %d] expected %+v, got %+v", idx, test.Error, err)
		}
	}
}

// errorContains checks if the error message in out contains the text in
// want.
//
// This is safe when out is nil. Use an empty string for want if you want to
// test that err is nil.
func errorContains(out error, want error) bool {
	if want == nil && out == nil {
		return true
	} else if want == nil || out == nil {
		return false
	}
	return strings.Contains(out.Error(), want.Error())
}
