package skyhook

import (
	"crypto/sha256"
	"fmt"
	"os"
)

type Dataset struct {
	ID int
	Name string

	// data or computed
	Type string

	DataType DataType

	// nil unless Type=computed
	Hash *string
}

type Item struct {
	Dataset Dataset
	Key string
	Ext string
	Format string
	Metadata string

	// nil to use default storage provider for LoadData / UpdateData
	Provider *string
	ProviderInfo *string
}

func (ds Dataset) Dirname() string {
	return fmt.Sprintf("items/%d", ds.ID)
}

func (ds Dataset) Mkdir() {
	os.Mkdir(ds.Dirname(), 0755)
}

func (item Item) Fname() string {
	provider := item.GetProvider()
	if provider.Fname == nil {
		return ""
	}
	return provider.Fname(item)
}

func (item Item) GetProvider() ItemProvider {
	if item.Provider == nil {
		return DefaultItemProvider
	} else {
		return ItemProviders[*item.Provider]
	}
}

func (item Item) UpdateData(data Data) {
	provider := item.GetProvider()
	if provider.UpdateData == nil {
		panic(fmt.Errorf("UpdateData not supported in dataset %s", item.Dataset.Name))
	}
	err := provider.UpdateData(item, data)
	if err != nil {
		// TODO
		panic(err)
	}
}

func (item Item) LoadData() (Data, error) {
	return item.GetProvider().LoadData(item)
}

func (ds Dataset) Remove() {
	os.RemoveAll(fmt.Sprintf("items/%d", ds.ID))
}

func (item Item) Remove() {
	fname := item.Fname()
	if fname == "" {
		panic(fmt.Errorf("Remove not supported in dataset %s", item.Dataset.Name))
	}
	os.Remove(fname)
}

// Copy the data to the specified filename with specified output format.
// If symlink is true, we try to symlink when possible.
// In some cases, copying data isn't possible and we need to actually load it (decode+re-encode).
func (item Item) CopyTo(fname string, format string, symlink bool) error {
	srcFname := item.Fname()
	if srcFname != "" && format == item.Format {
		return CopyOrSymlink(srcFname, fname, symlink)
	}

	// so either the file is not directly available, or the format doesn't match
	// either way, we need to load the data and re-encode it
	data, err := item.LoadData()
	if err != nil {
		return err
	}
	file, err := os.Create(fname)
	if err != nil {
		return err
	}
	defer file.Close()
	return data.Encode(format, file)
}

func (ds Dataset) DBFname() string {
	return fmt.Sprintf("items/%d/db.sqlite3", ds.ID)
}

func (ds Dataset) LocalHash() []byte {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("name=%s\n", ds.Name)))
	return h.Sum(nil)
}

type ItemProvider struct {
	LoadData func(item Item) (Data, error)
	// optional: we panic if UpdateData is called without being supported
	UpdateData func(item Item, data Data) error
	// optional: we return empty string if Fname is called without being supported
	// caller then needs to fallback to loading the data
	Fname func(item Item) string
}

var ItemProviders = make(map[string]ItemProvider)

var DefaultItemProvider ItemProvider

// helper function to create virtual item providers
// virtual providers reference another dataset, calling LoadData on the other dataset
// but then applying some function on the data before returning it
// in virtual providers, ProviderInfo is JSON-encoded item in other dataset
// TODO: but this means stack of virtual providers will make item metadata keep getting longer and longer...
func VirtualProvider(f func(item Item, data Data) (Data, error), visibleFname bool) ItemProvider{
	var provider ItemProvider
	provider.LoadData = func(item Item) (Data, error) {
		var wrappedItem Item
		JsonUnmarshal([]byte(*item.ProviderInfo), &wrappedItem)
		data, err := wrappedItem.LoadData()
		if err != nil {
			return nil, err
		}
		return f(item, data)
	}
	provider.Fname = func(item Item) string {
		if !visibleFname {
			return ""
		}
		var wrappedItem Item
		JsonUnmarshal([]byte(*item.ProviderInfo), &wrappedItem)
		return wrappedItem.Fname()
	}
	return provider
}

func init() {
	DefaultItemProvider = ItemProvider{
		LoadData: func(item Item) (Data, error) {
			return DecodeFile(item.Dataset.DataType, item.Format, item.Metadata, item.Fname())
		},
		UpdateData: func(item Item, data Data) error {
			item.Dataset.Mkdir()
			file, err := os.Create(item.Fname())
			if err != nil {
				return err
			}
			if err := data.Encode(item.Format, file); err != nil {
				return err
			}
			return nil
		},
		Fname: func(item Item) string {
			return fmt.Sprintf("items/%d/%s.%s", item.Dataset.ID, item.Key, item.Ext)
		},
	}

	// Supports items that reference another item, which may be in another dataset.
	// We currently implement the reference by filename.
	// Metadata is taken from the new item. So it could be different from the original metadata.
	ItemProviders["reference"] = ItemProvider{
		LoadData: func(item Item) (Data, error) {
			filename := *item.ProviderInfo
			return DecodeFile(item.Dataset.DataType, item.Format, item.Metadata, filename)
		},
		Fname: func(item Item) string {
			return *item.ProviderInfo
		},
	}
}
