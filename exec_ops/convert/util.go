package convert

import (
	"github.com/skyhookml/skyhookml/skyhook"
)

// Takes items in a File dataset and returns map from filename to item.
func ItemsToFileMap(items []skyhook.Item) map[string]skyhook.Item {
	m := make(map[string]skyhook.Item)
	for _, item := range items {
		var metadata skyhook.FileMetadata
		skyhook.JsonUnmarshal([]byte(item.Metadata), &metadata)
		m[metadata.Filename] = item
	}
	return m
}
