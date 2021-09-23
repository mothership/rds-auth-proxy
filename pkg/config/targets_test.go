package config_test

import (
	"testing"

	. "github.com/mothership/rds-auth-proxy/pkg/config"
)

func TestTargetGetHost(t *testing.T) {
	cases := []struct {
		Target       ProxyTarget
		ExpectedHost string
	}{
		{
			Target: ProxyTarget{
				Host: "0.0.0.0:8000",
			},
			ExpectedHost: "0.0.0.0:8000",
		},
		{
			Target: ProxyTarget{
				Host: "0.0.0.0:8000",
				PortForward: &PortForward{
					LocalPort: strPtr("8001"),
				},
			},
			ExpectedHost: "0.0.0.0:8001",
		},
	}

	for idx, test := range cases {
		result := test.Target.GetHost()
		if result != test.ExpectedHost {
			t.Errorf("[Case %d] Expected %q, got %q", idx, test.ExpectedHost, result)
		}
	}
}

func TestTargetIsPortForward(t *testing.T) {
	cases := []struct {
		Target   ProxyTarget
		Expected bool
	}{
		{
			Target: ProxyTarget{
				Host: "0.0.0.0:8000",
			},
			Expected: false,
		},
		{
			Target: ProxyTarget{
				Host: "0.0.0.0:8000",
				PortForward: &PortForward{
					LocalPort: strPtr("8001"),
				},
			},
			Expected: true,
		},
	}

	for idx, test := range cases {
		result := test.Target.IsPortForward()
		if result != test.Expected {
			t.Errorf("[Case %d] Expected %t, got %t", idx, test.Expected, result)
		}
	}
}
