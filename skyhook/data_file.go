package skyhook

import (
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
)

type FileMetadata struct {
	Filename string
}

type FileHeader struct {
	FileMetadata
	Length int
}

type FileData struct {
	Bytes []byte
	Metadata FileMetadata
}

func (d FileData) EncodeStream(w io.Writer) error {
	WriteJsonData(FileHeader{
		FileMetadata: d.Metadata,
		Length: len(d.Bytes),
	}, w)
	if _, err := w.Write(d.Bytes); err != nil {
		return err
	}
	return nil
}

func (d FileData) Encode(format string, w io.Writer) error {
	_, err := w.Write(d.Bytes)
	return err
}

func (d FileData) Type() DataType {
	return FileType
}

func (d FileData) GetDefaultExtAndFormat() (string, string) {
	ext := filepath.Ext(d.Metadata.Filename)
	if len(ext) > 0 {
		ext = ext[1:]
	}
	return ext, ""
}

func (d FileData) GetMetadata() interface{} {
	return d.Metadata
}

func init() {
	DataImpls[FileType] = DataImpl{
		DecodeStream: func(r io.Reader) (Data, error) {
			var header FileHeader
			if err := ReadJsonData(r, &header); err != nil {
				return nil, err
			}
			bytes := make([]byte, header.Length)
			if _, err := io.ReadFull(r, bytes); err != nil {
				return nil, err
			}
			return FileData{
				Metadata: header.FileMetadata,
				Bytes: bytes,
			}, nil
		},
		Decode: func(format string, metadataRaw string, r io.Reader) (Data, error) {
			var metadata FileMetadata
			JsonUnmarshal([]byte(metadataRaw), &metadata)

			bytes, err := ioutil.ReadAll(r)
			if err != nil {
				return nil, err
			}

			return FileData{
				Metadata: metadata,
				Bytes: bytes,
			}, nil
		},
		GetDefaultMetadata: func(fname string) (format string, metadataRaw string, err error) {
			return "", "", fmt.Errorf("file metadata cannot be determined from file")
		},
	}
}
