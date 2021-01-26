package skyhook

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

type DataType string
const (
	ImageType DataType = "image"
	VideoType = "video"
	DetectionType = "detection"
	TrackType = "track"
	ShapeType = "shape"
	IntType = "int"
	FloatsType = "floats"
	ImListType = "imlist"
	TextType = "text"
	StringType = "string"
)

var DataTypes = map[string]DataType{
	"Image": ImageType,
	"Video": VideoType,
	"Detection": DetectionType,
	"Track": TrackType,
	"Shape": ShapeType,
	"Int": IntType,
	"Floats": FloatsType,
	"Image List": ImListType,
	"Text": TextType,
	"String": StringType,
}

func EncodeTypes(types []DataType) string {
	strs := make([]string, len(types))
	for i, t := range types {
		strs[i] = string(t)
	}
	return strings.Join(strs, ",")
}

func DecodeTypes(s string) []DataType {
	strs := strings.Split(s, ",")
	var types []DataType
	for _, str := range strs {
		if str == "" {
			continue
		}
		types = append(types, DataType(str))
	}
	return types
}

type DataImpl struct {
	DecodeStream func(r io.Reader) (Data, error)
	Decode func(format string, metadata string, r io.Reader) (Data, error)
	GetDefaultMetadata func(fname string) (format string, metadata string, err error)
	DefaultFormat string

	// optional: if not set, caller should call Decode
	DecodeFile func(format string, metadata string, fname string) (Data, error)

	// optional: some data types may not support this
	Builder func() ChunkBuilder
	ChunkType DataType
}

var DataImpls = make(map[DataType]DataImpl)

type Data interface {
	EncodeStream(w io.Writer) error
	Encode(format string, w io.Writer) error
	Type() DataType
	GetMetadata() interface{}
	GetDefaultExtAndFormat() (ext string, format string)
}

func DecodeData(t DataType, format string, metadata string, r io.Reader) (Data, error) {
	return DataImpls[t].Decode(format, metadata, r)
}

func DecodeFile(t DataType, format string, metadata string, fname string) (Data, error) {
	impl := DataImpls[t]
	if impl.DecodeFile != nil {
		return impl.DecodeFile(format, metadata, fname)
	}
	file, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return impl.Decode(format, metadata, file)
}

func GetDefaultFormat(t DataType) string {
	return DataImpls[t].DefaultFormat
}

func WriteJsonData(x interface{}, w io.Writer) error {
	bytes := JsonMarshal(x)
	blen := make([]byte, 4)
	binary.BigEndian.PutUint32(blen, uint32(len(bytes)))
	w.Write(blen)
	_, err := w.Write(bytes)
	return err
}

func ReadJsonData(r io.Reader, x interface{}) error {
	blen := make([]byte, 4)
	if _, err := io.ReadFull(r, blen); err != nil {
		return err
	}
	bytes := make([]byte, binary.BigEndian.Uint32(blen))
	if _, err := io.ReadFull(r, bytes); err != nil {
		return err
	}
	return json.Unmarshal(bytes, x)
}

type SliceData interface {
	Data
	Length() int
	Slice(i, j int) Data
	Append(other Data) Data
}

type ReadableData interface {
	Data
	Reader() DataReader
}

type DataReader interface {
	Read(n int) (Data, error)
	Close()
}

// Chunk builder enables creating a Data from chunks of some SliceData.
// For example, VideoData supports building from chunks of ImageData.
// Generally SliceData types X support building from chunks of X (itself) using SliceBuilder.
type ChunkBuilder interface {
	Write(chunk Data) error
	Close() (Data, error)
}

type SliceReader struct {
	Data SliceData
	pos int
}

func (r *SliceReader) Read(n int) (Data, error) {
	remaining := r.Data.Length() - r.pos
	if remaining <= 0 {
		return nil, io.EOF
	}
	if remaining < n {
		n = remaining
	}
	data := r.Data.Slice(r.pos, r.pos+n)
	r.pos += n
	return data, nil
}

func (r *SliceReader) Close() {}

type SliceBuilder struct {
	Data SliceData
}

func (b *SliceBuilder) Write(chunk Data) error {
	b.Data = b.Data.Append(chunk).(SliceData)
	return nil
}

func (b *SliceBuilder) Close() (Data, error) {
	return b.Data, nil
}

func SynchronizedReader(inputs []Data, n int, f func(pos int, length int, datas []Data) error) error {
	readers := make([]DataReader, len(inputs))
	for i, input := range inputs {
		readers[i] = input.(ReadableData).Reader()
	}

	defer func() {
		for _, rd := range readers {
			rd.Close()
		}
	}()

	pos := 0
	for {
		datas := make([]Data, len(inputs))
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
			length := data.(SliceData).Length()
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

func PerFrame(inputs []Data, f func(pos int, datas []Data) error) error {
	SynchronizedReader(inputs, 32, func(pos int, length int, datas []Data) error {
		for i := 0; i < length; i++ {
			var cur []Data
			for _, d := range datas {
				cur = append(cur, d.(SliceData).Slice(i, i+1))
			}
			err := f(pos+i, cur)
			if err != nil {
				return err
			}
		}
		return nil
	})
	return nil
}
