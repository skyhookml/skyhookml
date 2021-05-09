package skyhook

import (
	"fmt"
	"io"
	"os"
)

// Read multiple sequence-type datas in a synchronized fashion, in chunks of length [n].
// [f] is a callback to pass each chunk of data to.
func SynchronizedReader(items []Item, n int, f func(pos int, length int, datas []interface{}) error) error {
	specs := make([]SequenceDataSpec, len(items))
	readers := make([]SequenceReader, len(items))
	for i, item := range items {
		spec := DataSpecs[item.Dataset.DataType].(SequenceDataSpec)
		specs[i] = spec

		fname := item.Fname()
		if fname == "" {
			// Item doesn't exist on disk.
			// So we need to load the whole thing from disk, and then use a SliceReader.
			data, _, err := item.LoadData()
			if err != nil {
				return err
			}
			readers[i] = &SliceReader{
				Data: data,
				Spec: spec,
			}
			continue
		}

		metadata := item.DecodeMetadata()
		if fileSpec, ok := spec.(FileSequenceDataSpec); ok {
			rd := fileSpec.FileReader(item.Format, metadata, fname)
			defer rd.Close()
			readers[i] = rd
		} else {
			file, err := os.Open(fname)
			if err != nil {
				return err
			}
			rd := spec.Reader(item.Format, metadata, file)
			rd = ClosingSequenceReader{
				ReadCloser: file,
				Reader: rd,
			}
			defer rd.Close()
			readers[i] = rd
		}
	}

	pos := 0
	for {
		datas := make([]interface{}, len(items))
		var count int
		for i, rd := range readers {
			data, err := rd.Read(n)
			if err == io.EOF {
				if i > 0 && count != 0 {
					return fmt.Errorf("inputs have different lengths")
				}
				continue
			} else if err != nil {
				return fmt.Errorf("error reading from input %d: %v", i, err)
			}
			length := specs[i].Length(data)
			if i == 0 {
				count = length
			} else if count != length {
				return fmt.Errorf("inputs have different lengths")
			}
			datas[i] = data
		}

		if count == 0 {
			break
		}

		err := f(pos, count, datas)
		if err != nil {
			return err
		}

		pos += count
	}

	return nil
}

// Read multiple sequence-type datas one by one and pass to the callback [f].
func PerFrame(items []Item, f func(pos int, datas []interface{}) error) error {
	specs := make([]SequenceDataSpec, len(items))
	for i, item := range items {
		specs[i] = DataSpecs[item.Dataset.DataType].(SequenceDataSpec)
	}
	return SynchronizedReader(items, 32, func(pos int, length int, datas []interface{}) error {
		for i := 0; i < length; i++ {
			var cur []interface{}
			for itemIdx, d := range datas {
				cur = append(cur, specs[itemIdx].Slice(d, i, i+1))
			}
			err := f(pos+i, cur)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// Read multiple datas and try to break them up into chunks of length n.
// If there are any non-sequence-type datas, then we pass all the datas to the callback in one call.
// In that case, the length passed to the call is -1.
func TrySynchronizedReader(items []Item, n int, f func(pos int, length int, datas []interface{}) error) error {
	allSequence := true
	for _, item := range items {
		spec := DataSpecs[item.Dataset.DataType]
		_, ok := spec.(SequenceDataSpec)
		allSequence = allSequence && ok
	}
	if allSequence {
		return SynchronizedReader(items, n, f)
	}

	datas := make([]interface{}, len(items))
	for i, item := range items {
		fname := item.Fname()
		if fname == "" {
			// If file doesn't exist, we need to load it directly through the item.
			data, _, err := item.LoadData()
			if err != nil {
				return err
			}
			datas[i] = data
			continue
		}

		metadata := item.DecodeMetadata()
		data, err := DecodeFile(item.Dataset.DataType, item.Format, metadata, fname)
		if err != nil {
			return err
		}
		datas[i] = data
	}
	return f(0, -1, datas)
}
