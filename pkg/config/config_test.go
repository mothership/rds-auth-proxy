package config_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	. "github.com/mothership/rds-auth-proxy/pkg/config"
	"github.com/mothership/rds-auth-proxy/pkg/pg"
)

func TestProxyConfigLoad(t *testing.T) {
	cases := []struct {
		FileName          string
		Error             error
		ExpectedInHostMap *string
	}{
		{
			FileName:          "foo",
			Error:             fmt.Errorf("Unsupported Config Type"),
			ExpectedInHostMap: nil,
		},
		{
			FileName:          "foo.yaml",
			Error:             fmt.Errorf("no such file or directory"),
			ExpectedInHostMap: nil,
		},
	}
	for idx, test := range cases {
		pcfg, err := LoadConfig(context.Background(), &mockRDSClient{}, &mockRedshiftClient{}, test.FileName)
		if !errorContains(err, test.Error) {
			t.Errorf("[Case %d] expected %+v, got %+v", idx, test.Error, err)
		}
		if test.ExpectedInHostMap != nil {
			_, ok := pcfg.HostMap[*test.ExpectedInHostMap]
			if !ok {
				t.Errorf("[Case %d] expected %+v in hostmap", idx, test.ExpectedInHostMap)
			}
		}
	}
}

func TestConfigInit(t *testing.T) {
	var cfg ConfigFile
	cfg.Init()

	if cfg.Targets == nil {
		t.Errorf("expected targets to be initialized")
	}
	if cfg.ProxyTargets == nil {
		t.Errorf("expected proxy targets to be initialized")
	}
	if cfg.HostMap == nil {
		t.Errorf("expected hostmap to be initialized")
	}
}

func TestTargetsGetDefaults(t *testing.T) {
	var cfg ConfigFile = ConfigFile{
		Proxy: Proxy{
			SSL: ServerSSL{
				ClientCertificatePath: strPtr("/app/cert.pem"),
				ClientPrivateKeyPath:  strPtr("/app/key.pem"),
			},
		},
		Targets: map[string]*Target{
			"empty": {Host: "0"},
			"override": {
				Host: "1",
				SSL: SSL{
					ClientCertificatePath: strPtr("/tls/cert.pem"),
					ClientPrivateKeyPath:  strPtr("/tls/key.pem"),
				},
			},
		},
	}
	cfg.Init()

	if len(cfg.HostMap) != 2 {
		t.Errorf("expected hostmap to have 2 entries")
	}

	if cfg.Targets["empty"].Name != "empty" {
		t.Errorf("Expected empty to have name populated")
	}

	if cfg.Targets["override"].Name != "override" {
		t.Errorf("Expected override to have name populated")
	}

	if cfg.Targets["empty"].SSL.Mode != pg.SSLRequired {
		t.Errorf("Expected empty to require SSL")
	}

	if cfg.Targets["empty"].SSL.ClientCertificatePath != cfg.Proxy.SSL.ClientCertificatePath {
		t.Errorf("Expected SSL cert to have taken value from parent")
	}

	if cfg.Targets["empty"].SSL.ClientPrivateKeyPath != cfg.Proxy.SSL.ClientPrivateKeyPath {
		t.Errorf("Expected SSL key to have taken value from parent")
	}
}

func TestProxyTargetsGetDefault(t *testing.T) {
	var cfg ConfigFile = ConfigFile{
		Proxy: Proxy{
			SSL: ServerSSL{
				ClientCertificatePath: strPtr("/app/cert.pem"),
				ClientPrivateKeyPath:  strPtr("/app/key.pem"),
			},
		},
		ProxyTargets: map[string]*ProxyTarget{
			"empty": {Host: "0"},
			"portforward": {
				PortForward: &PortForward{},
			},
			"override": {
				Host: "1",
				SSL: SSL{
					ClientCertificatePath: strPtr("/tls/cert.pem"),
					ClientPrivateKeyPath:  strPtr("/tls/key.pem"),
				},
			},
		},
	}
	cfg.Init()

	if cfg.ProxyTargets["empty"].Name != "empty" {
		t.Errorf("Expected empty to have name populated")
	}

	if cfg.ProxyTargets["override"].Name != "override" {
		t.Errorf("Expected override to have name populated")
	}

	if cfg.ProxyTargets["empty"].SSL.Mode != pg.SSLRequired {
		t.Errorf("Expected empty to require SSL")
	}

	if cfg.ProxyTargets["empty"].SSL.ClientCertificatePath != cfg.Proxy.SSL.ClientCertificatePath {
		t.Errorf("Expected SSL cert to have taken value from parent")
	}

	if cfg.ProxyTargets["empty"].SSL.ClientPrivateKeyPath != cfg.Proxy.SSL.ClientPrivateKeyPath {
		t.Errorf("Expected SSL key to have taken value from parent")
	}

	if cfg.ProxyTargets["portforward"].SSL.Mode != pg.SSLDisabled {
		t.Errorf("Expected SSL to be disabled on portforward if not set")
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
