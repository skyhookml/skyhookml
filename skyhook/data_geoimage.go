package skyhook

import (
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
	ReferenceType string `json:",omitempty"`

	// For "webmercator" georeference type.
	// Zoom, X, and Y specify the cell that the image spans.
	Zoom int `json:",omitempty"`
	X int `json:",omitempty"`
	Y int `json:",omitempty"`
	// Scale specifies resolution of each cell. Usually it is 256.
	Scale int `json:",omitempty"`
	// Offset specifies offset from the X,Y tile.
	Offset [2]int `json:",omitempty"`

	// Width and height corresponding to the specified zoom and scale.
	// This does not necessarily match the image width and height, but if image
	// is resized from this width and height then the x / y axis must still be proportional.
	Width int `json:",omitempty"`
	Height int `json:",omitempty"`

	// For custom formats, optionally store the longitude-latitude of bottom-left and top-right corners.
	// If set, we assume that the projection is like Mercator where compass direction matches image axes.
	// If not set, we cannot transform between longitude-latitude and pixel coordinates.
	Bbox [4]float64 `json:",omitempty"`

	// image source type
	// "local": image is stored in JPEG file
	// "url": image comes from a tile server
	// "dataset": image comes from another dataset
	SourceType string `json:",omitempty"`

	// For URL type, the tile server URL.
	URL string `json:",omitempty"`

	// For dataset type, define the source items.
	Items []GeoImageItemSource `json:",omitempty"`
}

func (m GeoImageMetadata) Update(other DataMetadata) DataMetadata {
	other_ := other.(GeoImageMetadata)
	if other_.ReferenceType != "" {
		m.ReferenceType = other_.ReferenceType
	}
	if other_.Zoom != 0 {
		m.Zoom = other_.Zoom
	}
	if other_.X != 0 {
		m.X = other_.X
	}
	if other_.Y != 0 {
		m.Y = other_.Y
	}
	if other_.Scale != 0 {
		m.Scale = other_.Scale
	}
	if other_.Offset[0] != 0 {
		m.Offset = other_.Offset
	}
	if other_.Width != 0 {
		m.Width = other_.Width
	}
	if other_.Height != 0 {
		m.Height = other_.Height
	}
	if other_.Bbox[0] != 0 {
		m.Bbox = other_.Bbox
	}
	if other_.SourceType != "" {
		m.SourceType = other_.SourceType
	}
	if other_.URL != "" {
		m.URL = other_.URL
	}
	if len(other_.Items) > 0 {
		m.Items = other_.Items
	}
	return m
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

type GeoImageDataSpec struct{}

func (s GeoImageDataSpec) DecodeMetadata(rawMetadata string) DataMetadata {
	if rawMetadata == "" {
		return GeoImageMetadata{}
	}
	var m GeoImageMetadata
	JsonUnmarshal([]byte(rawMetadata), &m)
	return m
}

func (s GeoImageDataSpec) ReadStream(r io.Reader) (interface{}, error) {
	var header ImageStreamHeader
	if err := ReadJsonData(r, &header); err != nil {
		return nil, err
	}
	if header.Length == 0 {
		return nil, nil
	}
	bytes := make([]byte, header.Width*header.Height*3)
	if _, err := io.ReadFull(r, bytes); err != nil {
		return nil, err
	}
	image := Image{
		Width: header.Width,
		Height: header.Height,
		Bytes: bytes,
	}
	return image, nil
}

func (s GeoImageDataSpec) WriteStream(data interface{}, w io.Writer) error {
	image := data.(Image)
	header := ImageStreamHeader{
		Width: image.Width,
		Height: image.Height,
		Channels: 3,
		Length: 1,
		BytesPerElement: len(image.Bytes),
	}
	if err := WriteJsonData(header, w); err != nil {
		return err
	}
	if _, err := w.Write(image.Bytes); err != nil {
		return err
	}
	return nil
}

func (s GeoImageDataSpec) Read(format string, metadata_ DataMetadata, r io.Reader) (data interface{}, err error) {
	// Check if image is available locally.
	if format == "jpeg" {
		image, err := ImageFromJPGReader(r)
		if err != nil {
			return nil, err
		}
		return image, nil
	}

	metadata := metadata_.(GeoImageMetadata)

	if metadata.SourceType == "url" {
		if metadata.ReferenceType != "webmercator" {
			return Image{}, fmt.Errorf("URL source type only supported for webmercator reference type")
		}

		// compute bounding box in global pixel coordinates
		sx := metadata.X * metadata.Scale + metadata.Offset[0]
		sy := metadata.Y * metadata.Scale + metadata.Offset[1]
		ex := sx + metadata.Width
		ey := sy + metadata.Height

		im := NewImage(metadata.Width, metadata.Height)

		// download each tile and stick it in the right place
		for i := sx/metadata.Scale; i <= (ex-1)/metadata.Scale; i++ {
			for j := sy/metadata.Scale; j <= (ey-1)/metadata.Scale; j++ {
				tile, err := metadata.DownloadTile(i, j)
				if err != nil {
					return Image{}, err
				}
				dstPos := [2]int{
					i*metadata.Scale - sx,
					j*metadata.Scale - sy,
				}
				im.DrawImage(dstPos[0], dstPos[1], tile)
			}
		}

		return im, nil
	} else if metadata.SourceType == "dataset" {
		im := NewImage(metadata.Width, metadata.Height)
		for _, srcItem := range metadata.Items {
			data, _, err := srcItem.Item.LoadData()
			if err != nil {
				return Image{}, fmt.Errorf("error loading source tile in dataset %d: %v", srcItem.Item.Dataset.ID, err)
			}
			im.DrawImage(srcItem.Offset[0], srcItem.Offset[1], data.(Image))
		}

		return im, nil
	}

	return nil, fmt.Errorf("unknown source type %s", metadata.SourceType)
}

func (s GeoImageDataSpec) Write(data interface{}, format string, metadata_ DataMetadata, w io.Writer) error {
	metadata := metadata_.(GeoImageMetadata)
	if format == "txt" {
		if metadata.SourceType != "url" && metadata.SourceType != "dataset" {
			return fmt.Errorf("cannot encode GeoImage with source type %s to txt format", metadata.SourceType)
		}
		// don't need to write anything since data is stored at the URL or dataset
		return nil
	} else if format == "jpeg" {
		if metadata.SourceType != "local" {
			return fmt.Errorf("only local source type can be encoded as jpeg")
		}
		image := data.(Image)
		bytes, err := image.AsJPG()
		if err != nil {
			return err
		}
		_, err = w.Write(bytes)
		return err
	}
	return fmt.Errorf("unknown format %s", format)
}

func (s GeoImageDataSpec) GetDefaultExtAndFormat(data interface{}, metadata_ DataMetadata) (ext string, format string) {
	metadata := metadata_.(GeoImageMetadata)
	if metadata.SourceType == "local" {
		return "jpg", "jpeg"
	} else {
		return "txt", "txt"
	}
}

func init() {
	DataSpecs[GeoImageType] = GeoImageDataSpec{}
}
