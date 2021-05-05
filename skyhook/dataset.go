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
	Metadata string

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
	return fmt.Sprintf("data/items/%d", ds.ID)
}

func (ds Dataset) Mkdir() {
	os.Mkdir(ds.Dirname(), 0755)
}

func (ds Dataset) DataSpec() DataSpec {
	return DataSpecs[ds.DataType]
}

func (item Item) DataSpec() DataSpec {
	return item.Dataset.DataSpec()
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

func (item Item) UpdateData(data interface{}, metadata DataMetadata) error {
	provider := item.GetProvider()
	if provider.UpdateData == nil {
		panic(fmt.Errorf("UpdateData not supported in dataset %s", item.Dataset.Name))
	}
	return provider.UpdateData(item, data, metadata)
}

func (item Item) LoadData() (interface{}, DataMetadata, error) {
	return item.GetProvider().LoadData(item)
}

func (item Item) LoadReader() (SequenceReader, DataMetadata) {
	metadata := item.DecodeMetadata()
	spec, ok := item.DataSpec().(SequenceDataSpec)
	if !ok {
		return ErrorSequenceReader{fmt.Errorf("data type %s is not sequence type", item.Dataset.DataType)}, metadata
	}

	fname := item.Fname()
	if fname == "" {
		// Since file is not available, we need to load the data and then return SliceReader.
		data, _, err := item.LoadData()
		if err != nil {
			return ErrorSequenceReader{err}, metadata
		}
		return &SliceReader{
			Data: data,
			Spec: spec,
		}, metadata
	}

	if fileSpec, fileOK := spec.(FileSequenceDataSpec); fileOK {
		return fileSpec.FileReader(item.Format, metadata, fname), metadata
	}
	file, err := os.Open(fname)
	if err != nil {
		return ErrorSequenceReader{err}, metadata
	}
	return ClosingSequenceReader{
		Reader: spec.Reader(item.Format, metadata, file),
		ReadCloser: file,
	}, nil
}

func (item Item) LoadWriter() SequenceWriter {
	metadata := item.DecodeMetadata()
	spec, ok := item.DataSpec().(SequenceDataSpec)
	if !ok {
		return ErrorSequenceWriter{fmt.Errorf("data type %s is not sequence type", item.Dataset.DataType)}
	}

	item.Dataset.Mkdir()
	fname := item.Fname()
	if fileSpec, fileOK := spec.(FileSequenceDataSpec); fileOK {
		return fileSpec.FileWriter(item.Format, metadata, fname)
	}
	file, err := os.Create(fname)
	if err != nil {
		return ErrorSequenceWriter{err}
	}
	return ClosingSequenceWriter{
		Writer: spec.Writer(item.Format, metadata, file),
		WriteCloser: file,
	}
}

func (ds Dataset) Remove() {
	os.RemoveAll(fmt.Sprintf("data/items/%d", ds.ID))
}

func (item Item) Remove() {
	fname := item.Fname()
	if fname == "" {
		panic(fmt.Errorf("Remove not supported in dataset %s", item.Dataset.Name))
	}
	os.Remove(fname)
}

func (item Item) DecodeMetadata() DataMetadata {
	spec := item.DataSpec()
	metadata := spec.DecodeMetadata(item.Dataset.Metadata)
	metadata = metadata.Update(spec.DecodeMetadata(item.Metadata))
	return metadata
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
	data, _, err := item.LoadData()
	if err != nil {
		return err
	}
	file, err := os.Create(fname)
	if err != nil {
		return err
	}
	defer file.Close()
	return item.DataSpec().Write(data, format, item.DecodeMetadata(), file)
}

func (ds Dataset) DBFname() string {
	return fmt.Sprintf("data/items/%d/db.sqlite3", ds.ID)
}

func (ds Dataset) LocalHash() []byte {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("name=%s\n", ds.Name)))
	return h.Sum(nil)
}

type ItemProvider struct {
	LoadData func(item Item) (interface{}, DataMetadata, error)
	// optional: we panic if UpdateData is called without being supported
	UpdateData func(item Item, data interface{}, metadata DataMetadata) error
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
func VirtualProvider(f func(item Item, data interface{}, metadata DataMetadata) (interface{}, DataMetadata, error), visibleFname bool) ItemProvider{
	var provider ItemProvider
	provider.LoadData = func(item Item) (interface{}, DataMetadata, error) {
		var wrappedItem Item
		JsonUnmarshal([]byte(*item.ProviderInfo), &wrappedItem)
		data, _, err := wrappedItem.LoadData()
		if err != nil {
			return nil, nil, err
		}
		return f(item, data, item.DecodeMetadata())
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
		LoadData: func(item Item) (interface{}, DataMetadata, error) {
			metadata := item.DecodeMetadata()
			data, err := DecodeFile(item.Dataset.DataType, item.Format, metadata, item.Fname())
			if err != nil {
				return nil, nil, fmt.Errorf("error reading item %s: %v", item.Key, err)
			}
			return data, metadata, nil
		},
		UpdateData: func(item Item, data interface{}, metadata DataMetadata) error {
			item.Dataset.Mkdir()
			file, err := os.Create(item.Fname())
			if err != nil {
				return err
			}
			defer file.Close()
			if err := item.DataSpec().Write(data, item.Format, metadata, file); err != nil {
				return err
			}
			return nil
		},
		Fname: func(item Item) string {
			return fmt.Sprintf("data/items/%d/%s.%s", item.Dataset.ID, item.Key, item.Ext)
		},
	}

	// Supports items that reference another item, which may be in another dataset.
	// We currently implement the reference by filename.
	// Metadata is taken from the new item. So it could be different from the original metadata.
	ItemProviders["reference"] = ItemProvider{
		LoadData: func(item Item) (interface{}, DataMetadata, error) {
			metadata := item.DecodeMetadata()
			filename := *item.ProviderInfo
			data, err := DecodeFile(item.Dataset.DataType, item.Format, metadata, filename)
			if err != nil {
				return nil, nil, err
			}
			return data, metadata, err
		},
		Fname: func(item Item) string {
			return *item.ProviderInfo
		},
	}
}
