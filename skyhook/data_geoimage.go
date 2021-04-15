package skyhook

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"strconv"
	"strings"

	gomapinfer "github.com/mitroadmaps/gomapinfer/common"
	geocoords "github.com/mitroadmaps/gomapinfer/googlemaps"
)

type GeoImageItemSource struct {
	Item Item
	// Copy the source image to this offset in our image.
	// Coordinates can be negative. For example, (-50, -50) means that (50, 50) in
	// the source item is placed at (0, 0) in our image.
	Offset [2]int
}

type GeoBbox [4]float64

// Convert from relative fractional position in image to longitude-latitude.
func (bbox GeoBbox) ToGeo(p [2]float64) [2]float64 {
	return [2]float64{
		bbox[0]+p[0]*(bbox[2]-bbox[0]),
		bbox[1]+(1-p[1])*(bbox[3]-bbox[1]),
	}
}

// Convert from longitude-latitude to fractional position in image.
func (bbox GeoBbox) FromGeo(p [2]float64) [2]float64 {
	return [2]float64{
		(p[0]-bbox[0])/(bbox[2]-bbox[0]),
		1-(p[1]-bbox[1])/(bbox[3]-bbox[1]),
	}
}

func (bbox GeoBbox) Rect() gomapinfer.Rectangle {
	return gomapinfer.Rectangle{
		Min: gomapinfer.Point{bbox[0], bbox[1]},
		Max: gomapinfer.Point{bbox[2], bbox[3]},
	}
}

type GeoImageMetadata struct {
	// either "webmercator" or "custom"
	ReferenceType string

	// For "webmercator" georeference type.
	// Zoom, X, and Y specify the cell that the image spans.
	Zoom int
	X int
	Y int
	// Scale specifies resolution of each cell. Usually it is 256.
	Scale int
	// Offset specifies offset from the X,Y tile.
	Offset [2]int

	// Width and height corresponding to the specified zoom and scale.
	// This does not necessarily match the image width and height, but if image
	// is resized from this width and height then the x / y axis must still be proportional.
	Width int
	Height int

	// For custom formats, optionally store the longitude-latitude of bottom-left and top-right corners.
	// If set, we assume that the projection is like Mercator where compass direction matches image axes.
	// If not set, we cannot transform between longitude-latitude and pixel coordinates.
	Bbox [4]float64

	// image source type
	// "local": image is stored in JPEG file
	// "url": image comes from a tile server
	// "dataset": image comes from another dataset
	SourceType string

	// For URL type, the tile server URL.
	URL string

	// For dataset type, define the source items.
	Items []GeoImageItemSource
}

// Assuming SourceType=="url", download a tile in this image from the URL.
func (m GeoImageMetadata) DownloadTile(i, j int) (Image, error) {
	url := m.URL
	url = strings.ReplaceAll(url, "[ZOOM]", strconv.Itoa(m.Zoom))
	url = strings.ReplaceAll(url, "[X]", strconv.Itoa(i))
	url = strings.ReplaceAll(url, "[Y]", strconv.Itoa(j))
	resp, err := http.Get(url)
	if err != nil {
		return Image{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errdesc string
		if !strings.HasPrefix(resp.Header.Get("Content-Type"), "image/") {
			if bytes, err := ioutil.ReadAll(resp.Body); err == nil {
				errdesc = string(bytes)
			}
		}
		return Image{}, fmt.Errorf("got status code %d (errdesc=%s)", resp.StatusCode, errdesc)
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType == "image/jpeg" {
		return ImageFromJPGReader(resp.Body)
	} else if contentType == "image/png" {
		return ImageFromPNGReader(resp.Body)
	}
	return Image{}, fmt.Errorf("unknown Content-Type %s", contentType)
}

// Get the bounding box of this image.
func (m GeoImageMetadata) GetBbox() GeoBbox {
	if m.ReferenceType == "custom" {
		return GeoBbox(m.Bbox)
	}

	// for webmercator, we use geocoords to convert to longitude-latitude
	originTile := [2]int{m.X, m.Y}
	offset := gomapinfer.Point{float64(m.Offset[0]), float64(m.Offset[1])}
	lengths := gomapinfer.Point{float64(m.Width), float64(m.Height)}
	p1 := geocoords.MapboxToLonLat(offset, m.Zoom, originTile)
	p2 := geocoords.MapboxToLonLat(offset.Add(lengths), m.Zoom, originTile)

	return GeoBbox{
		math.Min(p1.X, p2.X),
		math.Min(p1.Y, p2.Y),
		math.Max(p1.X, p2.X),
		math.Max(p1.Y, p2.Y),
	}
}

type GeoImageData struct {
	Metadata GeoImageMetadata
	Image Image
}

func (d GeoImageData) GetImage() (Image, error) {
	if len(d.Image.Bytes) > 0 {
		return d.Image, nil
	}

	if d.Metadata.SourceType == "url" {
		if d.Metadata.ReferenceType != "webmercator" {
			return Image{}, fmt.Errorf("URL source type only supported for webmercator reference type")
		}

		// compute bounding box in global pixel coordinates
		sx := d.Metadata.X * d.Metadata.Scale + d.Metadata.Offset[0]
		sy := d.Metadata.Y * d.Metadata.Scale + d.Metadata.Offset[1]
		ex := sx + d.Metadata.Width
		ey := sy + d.Metadata.Height

		im := NewImage(d.Metadata.Width, d.Metadata.Height)

		// download each tile and stick it in the right place
		for i := sx/d.Metadata.Scale; i <= (ex-1)/d.Metadata.Scale; i++ {
			for j := sy/d.Metadata.Scale; j <= (ey-1)/d.Metadata.Scale; j++ {
				tile, err := d.Metadata.DownloadTile(i, j)
				if err != nil {
					return Image{}, err
				}
				dstPos := [2]int{
					i*d.Metadata.Scale - sx,
					j*d.Metadata.Scale - sy,
				}
				im.DrawImage(dstPos[0], dstPos[1], tile)
			}
		}

		return im, nil
	} else if d.Metadata.SourceType == "dataset" {
		im := NewImage(d.Metadata.Width, d.Metadata.Height)
		for _, srcItem := range d.Metadata.Items {
			data_, err := srcItem.Item.LoadData()
			if err != nil {
				return Image{}, fmt.Errorf("error loading source tile in dataset %d: %v", srcItem.Item.Dataset.ID, err)
			}
			data := data_.(GeoImageData)
			tile, err := data.GetImage()
			if err != nil {
				return Image{}, fmt.Errorf("error loading source tile in dataset %d: %v", srcItem.Item.Dataset.ID, err)
			}
			im.DrawImage(srcItem.Offset[0], srcItem.Offset[1], tile)
		}

		return im, nil
	}

	return Image{}, fmt.Errorf("unknown source type %s", d.Metadata.SourceType)
}

type GeoImageStreamHeader struct {
	Metadata GeoImageMetadata
	Width int
	Height int
}

func (d GeoImageData) EncodeStream(w io.Writer) error {
	im, err := d.GetImage()
	if err != nil {
		return err
	}
	err = WriteJsonData(GeoImageStreamHeader{
		Metadata: d.Metadata,
		Width: im.Width,
		Height: im.Height,
	}, w)
	if err != nil {
		return err
	}
	_, err = w.Write(im.Bytes)
	if err != nil {
		return err
	}
	return nil
}

func (d GeoImageData) Encode(format string, w io.Writer) error {
	if format == "txt" {
		if d.Metadata.SourceType != "url" && d.Metadata.SourceType != "dataset" {
			return fmt.Errorf("cannot encode GeoImage with source type %s to txt format", d.Metadata.SourceType)
		}
		// don't need to write anything since data is stored at the URL or dataset
		return nil
	} else if format == "jpeg" {
		if d.Metadata.SourceType != "local" {
			return fmt.Errorf("only local source type can be encoded as jpeg")
		}
		bytes, err := d.Image.AsJPG()
		if err != nil {
			return err
		}
		_, err = w.Write(bytes)
		return err
	}
	return fmt.Errorf("unknown format %s", format)
}

func (d GeoImageData) Type() DataType {
	return GeoImageType
}

func (d GeoImageData) GetDefaultExtAndFormat() (string, string) {
	if d.Metadata.SourceType == "local" {
		return "jpg", "jpeg"
	} else {
		return "txt", "txt"
	}
}

func (d GeoImageData) GetMetadata() interface{} {
	return d.Metadata
}

func init() {
	DataImpls[GeoImageType] = DataImpl{
		DecodeStream: func(r io.Reader) (Data, error) {
			var header GeoImageStreamHeader
			if err := ReadJsonData(r, &header); err != nil {
				return nil, err
			}
			bytes := make([]byte, header.Width*header.Height*3)
			if _, err := io.ReadFull(r, bytes); err != nil {
				return nil, err
			}
			im := Image{
				Width: header.Width,
				Height: header.Height,
				Bytes: bytes,
			}
			return GeoImageData{
				Metadata: header.Metadata,
				Image: im,
			}, nil
		},
		Decode: func(format string, metadataRaw string, r io.Reader) (Data, error) {
			var metadata GeoImageMetadata
			if err := json.Unmarshal([]byte(metadataRaw), &metadata); err != nil {
				return nil, err
			}
			data := GeoImageData{Metadata: metadata}
			if format == "jpeg" {
				var err error
				data.Image, err = ImageFromJPGReader(r)
				if err != nil {
					return nil, err
				}
			}
			return data, nil
		},
		GetDefaultMetadata: func(fname string) (format string, metadataRaw string, err error) {
			return "", "", fmt.Errorf("GetDefaultMetadata not supported for GeoImage datasets")
		},
	}
}
