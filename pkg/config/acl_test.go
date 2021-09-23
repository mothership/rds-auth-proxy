package config_test

import (
	"testing"

	. "github.com/mothership/rds-auth-proxy/pkg/config"
)

func TestACLInit(t *testing.T) {
	var acl ACL
	acl.Init()
	if acl.AllowedRDSTags == nil {
		t.Errorf("Expected allowed tags not to be nil")
	}

	if acl.BlockedRDSTags == nil {
		t.Errorf("Expected blocked tags not to be nil")
	}
}
