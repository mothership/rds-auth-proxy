package config

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/rds/types"
)

// ACL represents rds instance tags allowed, or blocked by the proxy
type ACL struct {
	AllowedRDSTags TagList `mapstructure:"allowed_rds_tags"`
	BlockedRDSTags TagList `mapstructure:"blocked_rds_tags"`
}

// Init finishes initializing the ACL struct
func (a *ACL) Init() {
	if a.AllowedRDSTags == nil {
		a.AllowedRDSTags = []*Tag{}
	}

	if a.BlockedRDSTags == nil {
		a.BlockedRDSTags = []*Tag{}
	}
}

// IsAllowed returns an error if the instance tags are either not allowed,
// or explicitly blocked.
func (a *ACL) IsAllowed(tagList []types.Tag) error {
	tags := map[string]string{}
	for _, t := range tagList {
		tags[*t.Key] = *t.Value
	}

	for _, matcher := range a.AllowedRDSTags {
		value, ok := tags[matcher.Name]
		if !ok {
			return fmt.Errorf("tag %q not found on instance", matcher.Name)
		}
		if value != matcher.Value {
			return fmt.Errorf("tag %q has wrong value %q (wanted: %q)", matcher.Name, value, matcher.Value)
		}
	}
	for _, matcher := range a.BlockedRDSTags {
		value, ok := tags[matcher.Name]
		if !ok {
			continue
		}
		if value == matcher.Value {
			return fmt.Errorf("blocked by tag %q (value: %q)", matcher.Name, value)
		}
	}
	return nil
}
