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
		data, err := item.LoadData()
		if err != nil {
			http.Error(w, "no such item", 404)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		err = data.EncodeStream(w)
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
		inputDatas := make([]skyhook.Data, len(items))
		for i := range inputDatas {
			var err error
			inputDatas[i], err = items[i].LoadData()
			if err != nil {
				http.Error(w, fmt.Sprintf("load data error: %v", err), 400)
				s.Close()
				return
			}
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		err := skyhook.TrySynchronizedReader(inputDatas, 32, func(pos int, length int, datas []skyhook.Data) error {
			for _, data := range datas {
				err := data.EncodeStream(w)
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
			}
			err := skyhook.ReadJsonData(r.Body, &metaPacket)
			if err != nil {
				return err
			}
			dataImpl := skyhook.DataImpls[metaPacket.Dataset.DataType]
			data, err := dataImpl.DecodeStream(r.Body)
			if err != nil {
				return err
			}
			return exec_ops.WriteItem(url, metaPacket.Dataset, metaPacket.Key, data)
		}()
		if err != nil {
			log.Printf("[/write-item] error writing data: %v", err)
			http.Error(w, fmt.Sprintf("error writing data: %v", err), 400)
			s.Close()
			return
		}
	})

	// Create items using Builder.
	// We need to support creating multiple items in parallel here, since Python's
	// typical libraries for HTTP requests do not handle performing multiple
	// requests in parallel.
	mux.HandleFunc("/build", func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			// Initialize the builders.
			var metas []struct{
				Dataset skyhook.Dataset
				Key string
			}
			err := skyhook.ReadJsonData(r.Body, &metas)
			if err != nil {
				return err
			}
			builders := make([]skyhook.ChunkBuilder, len(metas))
			for i, meta := range metas {
				dataImpl := skyhook.DataImpls[meta.Dataset.DataType]
				builders[i] = dataImpl.Builder()
			}

			// Read and add to builders until EOF.
			eof := false
			for !eof {
				for i, meta := range metas {
					dataImpl := skyhook.DataImpls[meta.Dataset.DataType]
					data, err := dataImpl.DecodeStream(r.Body)
					if err == io.EOF {
						eof = true
						break
					} else if err != nil {
						return err
					}
					if err := builders[i].Write(data); err != nil {
						return err
					}
				}
			}

			// Add the extracted items to their respective datasets.
			for i, meta := range metas {
				data, err := builders[i].Close()
				if err != nil {
					return err
				}
				err = exec_ops.WriteItem(url, meta.Dataset, meta.Key, data)
				if err != nil {
					return err
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
