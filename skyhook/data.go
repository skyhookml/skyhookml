package skyhook

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"os"
	"strings"
)

type DataType string
const (
	ImageType DataType = "image"
	VideoType = "video"
	DetectionType = "detection"
	ShapeType = "shape"
	IntType = "int"
	FloatsType = "floats"
	ImListType = "imlist"
	TextType = "text"
	StringType = "string"
	ArrayType = "array"
	FileType = "file"
	TableType = "table"
	GeoImageType = "geoimage"
	GeoJsonType = "geojson"
)

var DataTypes = map[DataType]string{
	ImageType: "Image",
	VideoType: "Video",
	DetectionType: "Detection",
	ShapeType: "Shape",
	IntType: "Int",
	FloatsType: "Floats",
	ImListType: "Image List",
	TextType: "Text",
	StringType: "String",
	ArrayType: "Array",
	FileType: "File",
	TableType: "Table",
	GeoImageType: "Geo-Image",
	GeoJsonType: "GeoJSON",
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

// Metadata can mostly be anything but must support Update.
type DataMetadata interface {
	// Produce new metadata where fields that are specified by other overwrite
	// fields in the current metadata.
	Update(other DataMetadata) DataMetadata
}

// Specifies a data type.
type DataSpec interface {
	// Decode JSON-encoded metadata into a metadata object.
	DecodeMetadata(rawMetadata string) DataMetadata
	// Read data that has been written via WriteStream.
	ReadStream(r io.Reader) (data interface{}, err error)
	// Write data for reading via ReadStream.
	WriteStream(data interface{}, w io.Writer) error

	// Read data from storage.
	// format is the Item.Format which data types can use to describe how the data is stored.
	Read(format string, metadata DataMetadata, r io.Reader) (data interface{}, err error)
	// Write data to storage.
	Write(data interface{}, format string, metadata DataMetadata, w io.Writer) error

	// Given some data, return a suggested file extension and format to store it with.
	GetDefaultExtAndFormat(data interface{}, metadata DataMetadata) (ext string, format string)
}

type FileDataSpec interface {
	DataSpec
	// Read data directly from a file.
	ReadFile(format string, metadata DataMetadata, fname string) (data interface{}, err error)
	// Write data directly to a file.
	WriteFile(data interface{}, format string, metadata DataMetadata, fname string) error
}

type MetadataFromFileDataSpec interface {
	DataSpec
	// Given a filename, which should correspond to an actual file stored on disk,
	// returns a suitable format and metadata for reading that file.
	GetMetadataFromFile(fname string) (format string, metadata DataMetadata, err error)
}

type ExtFromFormatDataSpec interface {
	DataSpec
	// Given a format, return the standard extension corresponding to the format.
	// If a DataSpec doesn't implement this function, callers should use the format
	// as the file extension.
	GetExtFromFormat(format string) (ext string)
}

// SequenceDataSpec describes sequence data types.
// These are any data types consisting of a sequence of elements.
// For example, Detections are sequences of []Detection, while videos are sequences
// of images.
type SequenceDataSpec interface {
	DataSpec

	// Initialize a SequenceReader for reading data from storage.
	// The SequenceReader should read the data chunk by chunk.
	Reader(format string, metadata DataMetadata, r io.Reader) SequenceReader
	// Initialize a SequenceWriter to write chunk by chunk to storage.
	Writer(format string, metadata DataMetadata, w io.Writer) SequenceWriter

	// Slice operations on the sequence data.
	Length(data interface{}) int
	Append(data interface{}, more interface{}) interface{}
	Slice(data interface{}, i int, j int) interface{}
}

// Sequence data types that want to have special functionality when reading from
// local disk can implement FileReader and FileWriter.
type FileSequenceDataSpec interface {
	SequenceDataSpec
	FileReader(format string, metadata DataMetadata, fname string) SequenceReader
	FileWriter(format string, metadata DataMetadata, fname string) SequenceWriter
}

type RandomAccessDataSpec interface {
	SequenceDataSpec
	// Initialize a SequenceReader that starts reading at index i, and reads up to index j.
	ReadSlice(format string, metadata DataMetadata, fname string, i, j int) SequenceReader
}

type SequenceReader interface {
	Read(n int) (interface{}, error)
	Close()
}

type SequenceWriter interface {
	Write(data interface{}) error
	Close() error
}

var DataSpecs = make(map[DataType]DataSpec)

func ReadData(t DataType, format string, metadata DataMetadata, r io.Reader) (data interface{}, err error) {
	return DataSpecs[t].Read(format, metadata, r)
}

func DecodeFile(t DataType, format string, metadata DataMetadata, fname string) (data interface{}, err error) {
	spec := DataSpecs[t]
	if fileSpec, ok := spec.(FileDataSpec); ok {
		return fileSpec.ReadFile(format, metadata, fname)
	}
	file, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return spec.Read(format, metadata, file)
}

// Write x with JSON-encoding to a stream.
// Before writing the JSON-encoded data, we write the length of the data.
func WriteJsonData(x interface{}, w io.Writer) error {
	bytes := JsonMarshal(x)
	blen := make([]byte, 4)
	binary.BigEndian.PutUint32(blen, uint32(len(bytes)))
	w.Write(blen)
	_, err := w.Write(bytes)
	return err
}

// Reads data that was written by WriteJsonData.
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

type NoMetadata struct{}
func (m NoMetadata) Update(other DataMetadata) DataMetadata { return m }

// Forwards to ExtFromFormatDataSpec.GetExtFromFormat if available.
func GetExtFromFormat(dtype DataType, format string) string {
	spec := DataSpecs[dtype]
	if extSpec, ok := spec.(ExtFromFormatDataSpec); ok {
		return extSpec.GetExtFromFormat(format)
	}
	return ""
}

// SequenceReader/SequenceWriter that return errors.
type ErrorSequenceReader struct {
	Error error
}
func (r ErrorSequenceReader) Read(n int) (interface{}, error) {
	return nil, r.Error
}
func (r ErrorSequenceReader) Close() {}
type ErrorSequenceWriter struct {
	Error error
}
func (w ErrorSequenceWriter) Write(data interface{}) error {
	return w.Error
}
func (w ErrorSequenceWriter) Close() error {
	return w.Error
}

// SequenceReader for sequence data that has already been read into memory.
type SliceReader struct {
	Data interface{}
	Spec SequenceDataSpec
	pos int
}

func (r *SliceReader) Read(n int) (interface{}, error) {
	remaining := r.Spec.Length(r.Data) - r.pos
	if remaining <= 0 {
		return nil, io.EOF
	}
	if remaining < n {
		n = remaining
	}
	data := r.Spec.Slice(r.Data, r.pos, r.pos+n)
	r.pos += n
	return data, nil
}

func (r *SliceReader) Close() {}

func NewSliceReader(spec SequenceDataSpec, format string, metadata DataMetadata, r io.Reader) SequenceReader {
	data, err := spec.Read(format, metadata, r)
	if err != nil {
		return ErrorSequenceReader{err}
	}
	return &SliceReader{
		Data: data,
		Spec: spec,
	}
}

// SequenceWriter that stores everything in-memory until Close.
type SliceWriter struct {
	Spec SequenceDataSpec
	Format string
	Metadata DataMetadata
	Writer io.Writer
	data interface{}
}

func (w *SliceWriter) Write(data interface{}) error {
	if w.data == nil {
		w.data = data
	} else {
		w.data = w.Spec.Append(w.data, data)
	}
	return nil
}

func (w *SliceWriter) Close() error {
	return w.Spec.Write(w.data, w.Format, w.Metadata, w.Writer)
}

// SequenceReader that closes an io.ReadCloser on Close.
type ClosingSequenceReader struct {
	ReadCloser io.ReadCloser
	Reader SequenceReader
}
func (r ClosingSequenceReader) Read(n int) (interface{}, error) {
	return r.Reader.Read(n)
}
func (r ClosingSequenceReader) Close() {
	r.Reader.Close()
	r.ReadCloser.Close()
}

// SequenceWriter that closses an io.WriteCloser on Close.
type ClosingSequenceWriter struct {
	WriteCloser io.WriteCloser
	Writer SequenceWriter
}
func (w ClosingSequenceWriter) Write(data interface{}) error {
	return w.Writer.Write(data)
}
func (w ClosingSequenceWriter) Close() error {
	err1 := w.Writer.Close()
	err2 := w.WriteCloser.Close()
	if err1 != nil {
		return err1
	} else if err2 != nil {
		return err2
	}
	return nil
}
