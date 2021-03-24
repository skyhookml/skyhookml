package convert

import (
	"github.com/skyhookml/skyhookml/skyhook"

	"path/filepath"
)

// Takes items in a File dataset and returns map from filename to item.
// If flatten is true, we strip directory names.
func ItemsToFileMap(items []skyhook.Item, flatten bool) map[string]skyhook.Item {
	m := make(map[string]skyhook.Item)
	for _, item := range items {
		var metadata skyhook.FileMetadata
		skyhook.JsonUnmarshal([]byte(item.Metadata), &metadata)
		fname := metadata.Filename
		if flatten {
			fname = filepath.Base(fname)
		}
		m[fname] = item
	}
	return m
}
