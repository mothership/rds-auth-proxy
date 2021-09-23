package config

// Tag is an RDS tag
type Tag struct {
	Name  string `mapstructure:"name"`
	Value string `mapstructure:"value"`
}

// TagList is a list of tags
type TagList []*Tag

// Find returns a tag by name
func (t TagList) Find(key string) *Tag {
	for _, tag := range t {
		if tag.Name == key {
			return tag
		}
	}
	return nil
}
