package skyhook

import (
	"database/sql"
	"encoding/csv"
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

type TableData struct {
	// Column specifications.
	Specs []ColumnSpec
	// Rows.
	Data [][]string
}

func (d TableData) EncodeStream(w io.Writer) error {
	return WriteJsonData(d, w)
}

func (d TableData) WriteSQLFile(fname string) error {
	db, err := sql.Open("sqlite3", fname)
	if err != nil {
		return err
	}
	defer db.Close()
	cols := make([]string, len(d.Specs))
	for i := range cols {
		spec := d.Specs[i]
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
	for _, row := range d.Data {
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

func (d TableData) Encode(format string, w io.Writer) error {
	if format == "json" {
		_, err := w.Write(JsonMarshal(d))
		return err
	} else if format == "csv" {
		csvw := csv.NewWriter(w)
		// write labels
		labels := make([]string, len(d.Specs))
		for i := range labels {
			labels[i] = d.Specs[i].Label
		}
		csvw.Write(labels)
		for _, row := range d.Data {
			csvw.Write(row)
		}
		csvw.Flush()
		return csvw.Error()
	} else if format == "sqlite3" {
		// we create the database first as temporary file on disk
		// and then read the bytes and write it back to w
		tmpFname := filepath.Join(os.TempDir(), fmt.Sprintf("%d.sqlite3", rand.Int()))
		defer os.Remove(tmpFname)
		err := d.WriteSQLFile(tmpFname)
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

func (d TableData) Type() DataType {
	return TableType
}

func (d TableData) GetDefaultExtAndFormat() (string, string) {
	return "json", "json"
}

func (d TableData) GetMetadata() interface{} {
	return nil
}

func init() {
	DataImpls[TableType] = DataImpl{
		DecodeStream: func(r io.Reader) (Data, error) {
			var data TableData
			if err := ReadJsonData(r, &data); err != nil {
				return nil, err
			}
			return data, nil
		},
		Decode: func(format string, metadataRaw string, r io.Reader) (Data, error) {
			if format == "json" {
				bytes, err := ioutil.ReadAll(r)
				if err != nil {
					return nil, err
				}
				var data TableData
				JsonUnmarshal(bytes, &data)
				return data, nil
			} else if format == "csv" {
				csvr := csv.NewReader(r)
				records, err := csvr.ReadAll()
				if err != nil {
					return nil, err
				}
				var d TableData
				for _, s := range records[0] {
					d.Specs = append(d.Specs, ColumnSpec{
						Label: s,
						Type: "string",
					})
				}
				d.Data = records[1:]
				return d, nil
			} else if format == "sqlite3" {
				return nil, fmt.Errorf("decoding sqlite3 is not supported")
			}
			return nil, fmt.Errorf("unknown format %s", format)
		},
		GetDefaultMetadata: func(fname string) (format string, metadataRaw string, err error) {
			ext := filepath.Ext(fname)
			if ext == ".json" {
				return "json", "", nil
			} else if ext == ".csv" {
				return "csv", "", nil
			} else if ext == ".sqlite3" {
				return "sqlite3", "", nil
			}
			return "", "", fmt.Errorf("unknown extension %s for table type", ext)
		},
		GetExtGivenFormat: func(format string) string {
			if format == "json" || format == "csv" || format == "sqlite3" {
				return format
			}
			return ""
		},
	}
}
