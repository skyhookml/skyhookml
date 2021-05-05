package python

// Local HTTP server for the Python script to access.

import (
	"github.com/skyhookml/skyhookml/skyhook"
	"github.com/skyhookml/skyhookml/exec_ops"

	"fmt"
	"io"
	"log"
	"net"
	"net/http"
)

type HttpServer struct {
	Port int
	Listener *net.TCPListener
}

func (s HttpServer) Close() {
	s.Listener.Close()
}

func NewHttpServer(url string, f func(*http.ServeMux) error) (HttpServer, error) {
	mux := http.NewServeMux()
	var s HttpServer

	// item.LoadData
	mux.HandleFunc("/load-data", func(w http.ResponseWriter, r *http.Request) {
		var item skyhook.Item
		if err := skyhook.ParseJsonRequest(w, r, &item); err != nil {
			return
		}
		data, _, err := item.LoadData()
		if err != nil {
			log.Printf("[/load-data] error loading data: %v", err)
			http.Error(w, err.Error(), 400)
			s.Close()
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		err = item.DataSpec().WriteStream(data, w)
		if err != nil {
			log.Printf("[/load-data] error serving item %s: %v", item.Key, err)
			s.Close()
		}
	})

	// SynchronizedReader
	mux.HandleFunc("/synchronized-reader", func(w http.ResponseWriter, r *http.Request) {
		var items []skyhook.Item
		if err := skyhook.ParseJsonRequest(w, r, &items); err != nil {
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		err := skyhook.TrySynchronizedReader(items, 32, func(pos int, length int, datas []interface{}) error {
			for i, data := range datas {
				err := items[i].DataSpec().WriteStream(data, w)
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			log.Printf("[/synchronized-reader] error: %v", err)
			s.Close()
		}
	})

	// WriteItem
	mux.HandleFunc("/write-item", func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			var metaPacket struct {
				Dataset skyhook.Dataset
				Key string
				Metadata string
			}
			err := skyhook.ReadJsonData(r.Body, &metaPacket)
			if err != nil {
				return fmt.Errorf("error reading meta packet: %v", err)
			}
			spec := metaPacket.Dataset.DataSpec()
			metadata := spec.DecodeMetadata(metaPacket.Metadata)
			data, err := spec.ReadStream(r.Body)
			if err != nil {
				return fmt.Errorf("error reading data: %v", err)
			}
			err = exec_ops.WriteItem(url, metaPacket.Dataset, metaPacket.Key, data, metadata)
			if err != nil {
				return fmt.Errorf("error writing item: %v", err)
			}
			return nil
		}()
		if err != nil {
			log.Printf("[/write-item] error writing data: %v", err)
			http.Error(w, fmt.Sprintf("error writing data: %v", err), 400)
			s.Close()
			return
		}
	})

	// Create items using SequenceWriter.
	// We need to support creating multiple items in parallel here, since Python's
	// typical libraries for HTTP requests do not handle performing multiple
	// requests in parallel.
	mux.HandleFunc("/build", func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			// Read metadata packet.
			var metas []struct{
				Dataset skyhook.Dataset
				Key string
				Metadata string
			}
			err := skyhook.ReadJsonData(r.Body, &metas)
			if err != nil {
				return fmt.Errorf("error reading meta packet: %v", err)
			}

			// Read and add to builders until EOF.
			// We initialize the writers after receiving the first datas.
			// This is so that we can use GetDefaultExtAndFormat to create the items.
			writers := make([]skyhook.SequenceWriter, len(metas))
			eof := false
			for !eof {
				for i, meta := range metas {
					spec := meta.Dataset.DataSpec()
					data, err := spec.ReadStream(r.Body)
					if i == 0 && err == io.EOF {
						eof = true
						break
					} else if err != nil {
						return fmt.Errorf("error reading data: %v", err)
					}

					if writers[i] == nil {
						metadata := spec.DecodeMetadata(meta.Metadata)
						ext, format := spec.GetDefaultExtAndFormat(data, metadata)
						item, err := exec_ops.AddItem(url, meta.Dataset, meta.Key, ext, format, metadata)
						if err != nil {
							return err
						}
						writers[i] = item.LoadWriter()
					}

					if err := writers[i].Write(data); err != nil {
						return fmt.Errorf("error writing data: %v", err)
					}
				}
			}

			// Close the writers.
			for _, writer := range writers {
				err := writer.Close()
				if err != nil {
					return fmt.Errorf("error closing writer: %v", err)
				}
			}
			return nil
		}()
		if err != nil {
			log.Printf("[/build] error building data: %v", err)
			http.Error(w, fmt.Sprintf("error building data: %v", err), 400)
			s.Close()
			return
		}
	})

	if f != nil {
		err := f(mux)
		if err != nil {
			return HttpServer{}, err
		}
	}

	ln, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1")})
	if err != nil {
		return HttpServer{}, err
	}
	port := ln.Addr().(*net.TCPAddr).Port
	log.Printf("[python-http] starting HTTP server on port %d", port)
	go http.Serve(ln, mux)
	s = HttpServer{
		Port: port,
		Listener: ln,
	}
	return s, nil
}
