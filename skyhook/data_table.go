package skyhook

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type ColumnSpec struct {
	Label string
	// data type, one of "string", "int", "float64"
	Type string
}

type TableMetadata struct {
	Columns []ColumnSpec `json:",omitempty"`
}

func (m TableMetadata) Update(other DataMetadata) DataMetadata {
	other_ := other.(TableMetadata)
	if len(other_.Columns) > 0 {
		m.Columns = other_.Columns
	}
	return m
}

// Rows of table.
type TableData [][]string

type TableDataSpec struct{}

func (s TableDataSpec) DecodeMetadata(rawMetadata string) DataMetadata {
	if rawMetadata == "" {
		return TableMetadata{}
	}
	var m TableMetadata
	JsonUnmarshal([]byte(rawMetadata), &m)
	return m
}

func (s TableDataSpec) ReadStream(r io.Reader) (interface{}, error) {
	var data TableData
	if err := ReadJsonData(r, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (s TableDataSpec) WriteStream(data interface{}, w io.Writer) error {
	if err := WriteJsonData(data, w); err != nil {
		return err
	}
	return nil
}

func (s TableDataSpec) Read(format string, metadata DataMetadata, r io.Reader) (data interface{}, err error) {
	if format == "json" {
		bytes, err := ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}
		var data TableData
		if err := json.Unmarshal(bytes, &data); err != nil {
			return nil, err
		}
		return data, nil
	} else if format == "csv" {
		csvr := csv.NewReader(r)
		records, err := csvr.ReadAll()
		if err != nil {
			return nil, err
		}
		// Skip first row since it is the header.
		// This is stored in metadata and we would've already read in GetMetadataFromFile.
		var data = TableData(records[1:])
		return data, nil
	} else if format == "sqlite3" {
		return nil, fmt.Errorf("decoding sqlite3 is not supported")
	}
	return nil, fmt.Errorf("unknown format %s", format)
}

func (d TableDataSpec) WriteSQLFile(data [][]string, metadata TableMetadata, fname string) error {
	db, err := sql.Open("sqlite3", fname)
	if err != nil {
		return err
	}
	defer db.Close()
	cols := make([]string, len(metadata.Columns))
	for i := range cols {
		spec := metadata.Columns[i]
		var t string
		if spec.Type == "int" {
			t = "INTEGER"
		} else if spec.Type == "float64" {
			t = "REAL"
		} else {
			t = "TEXT"
		}
		cols[i] = spec.Label + " " + t
	}
	_, err = db.Exec("CREATE TABLE t (" + strings.Join(cols, ", ") + ")")
	if err != nil {
		return err
	}
	for _, row := range data {
		qs := make([]string, len(row))
		for i := range qs {
			qs[i] = "?"
		}
		var rowGeneric []interface{}
		for _, x := range row {
			rowGeneric = append(rowGeneric, x)
		}
		_, err = db.Exec("INSERT INTO t VALUES (" + strings.Join(qs, ", ") + ")", rowGeneric...)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s TableDataSpec) Write(data interface{}, format string, metadata_ DataMetadata, w io.Writer) error {
	rows := data.([][]string)
	metadata := metadata_.(TableMetadata)

	if format == "json" {
		_, err := w.Write(JsonMarshal(rows))
		return err
	} else if format == "csv" {
		csvw := csv.NewWriter(w)
		// Write column labels as first row in the CSV file.
		labels := make([]string, len(metadata.Columns))
		for i := range labels {
			labels[i] = metadata.Columns[i].Label
		}
		csvw.Write(labels)
		for _, row := range rows {
			csvw.Write(row)
		}
		csvw.Flush()
		return csvw.Error()
	} else if format == "sqlite3" {
		// we create the database first as temporary file on disk
		// and then read the bytes and write it back to w
		tmpFname := filepath.Join(os.TempDir(), fmt.Sprintf("%d.sqlite3", rand.Int()))
		defer os.Remove(tmpFname)
		err := s.WriteSQLFile(rows, metadata, tmpFname)
		if err != nil {
			return fmt.Errorf("error writing as sqlite3", err)
		}
		bytes, err := ioutil.ReadFile(tmpFname)
		if err != nil {
			return err
		}
		_, err = w.Write(bytes)
		if err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("unknown format %s", format)
}

func (s TableDataSpec) GetDefaultExtAndFormat(data interface{}, metadata DataMetadata) (ext string, format string) {
	return "json", "json"
}

func (s TableDataSpec) GetMetadataFromFile(fname string) (format string, metadata DataMetadata, err error) {
	ext := filepath.Ext(fname)
	if ext == ".json" {
		return "", nil, fmt.Errorf("metadata cannot be extracted from JSON format")
	} else if ext == ".csv" {
		file, err := os.Open(fname)
		if err != nil {
			return "", nil, err
		}
		defer file.Close()
		csvr := csv.NewReader(file)
		record, err := csvr.Read()
		if err != nil {
			return "", nil, err
		}
		var tmeta TableMetadata
		for _, s := range record {
			tmeta.Columns = append(tmeta.Columns, ColumnSpec{
				Label: s,
				Type: "string",
			})
		}
		return "csv", tmeta, nil
	} else if ext == ".sqlite3" {
		return "", nil, fmt.Errorf("decoding sqlite3 is not supported")
	}
	return "", nil, fmt.Errorf("unknown extension %s for table type", ext)
}

func init() {
	DataSpecs[TableType] = TableDataSpec{}
}
