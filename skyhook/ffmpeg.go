package skyhook

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
)

type VideoReader interface {
	// Error should be io.EOF if there are no more images.
	// If an image is returned, error must NOT be io.EOF.
	// (So no error should be returned on the last image, only after the last image.)
	Read() (Image, error)

	Close()
}

type FfmpegReader struct {
	Cmd *Cmd
	Stdout io.ReadCloser
	Width int
	Height int
	Buf []byte
}

func ReadFfmpeg(fname string, dims [2]int, rate [2]int) *FfmpegReader {
	log.Printf("[ffmpeg] from %s extract frames %dx%d", fname, dims[0], dims[1])

	cmd := Command(
		"ffmpeg-read", CommandOptions{NoStdin: true, OnlyDebug: true},
		"ffmpeg",
		"-threads", "2",
		"-i", fname,
		"-c:v", "rawvideo", "-pix_fmt", "rgb24", "-f", "rawvideo",
		"-vf", fmt.Sprintf("scale=%dx%d,fps=fps=%d/%d", dims[0], dims[1], rate[0], rate[1]),
		"-",
	)

	return &FfmpegReader{
		Cmd: cmd,
		Stdout: cmd.Stdout(),
		Width: dims[0],
		Height: dims[1],
		Buf: make([]byte, dims[0]*dims[1]*3),
	}
}

func (rd *FfmpegReader) Read() (Image, error) {
	_, err := io.ReadFull(rd.Stdout, rd.Buf)
	if err != nil {
		return Image{}, err
	}
	buf := make([]byte, len(rd.Buf))
	copy(buf, rd.Buf)
	im := ImageFromBytes(rd.Width, rd.Height, buf)
	return im, nil
}

func (rd *FfmpegReader) Close() {
	rd.Stdout.Close()
	rd.Cmd.Wait()
}

type ChanReader struct {
	Ch chan Image
}
func (rd *ChanReader) Read() (Image, error) {
	im, ok := <- rd.Ch
	if !ok {
		return Image{}, io.EOF
	}
	return im, nil
}
func (rd *ChanReader) Close() {
	go func() {
		for _ = range rd.Ch {}
	}()
}

func MakeVideo(rd VideoReader, dims [2]int, rate [2]int) (io.ReadCloser, *Cmd) {
	log.Printf("[ffmpeg] make video (%dx%d)", dims[0], dims[1])

	cmd := Command(
		"ffmpeg-mkvid", CommandOptions{OnlyDebug: true},
		"ffmpeg",
		"-threads", "2",
		"-f", "rawvideo",
		"-s", fmt.Sprintf("%dx%d", dims[0], dims[1]),
		"-r", fmt.Sprintf("%d/%d", rate[0], rate[1]),
		"-pix_fmt", "rgb24", "-i", "-",
		"-vcodec", "libx264", "-preset", "ultrafast", "-tune", "zerolatency", "-g", "30",
		"-vf", fmt.Sprintf("fps=fps=%d/%d", rate[0], rate[1]),
		"-f", "mp4", "-pix_fmt", "yuv420p", "-movflags", "faststart+frag_keyframe+empty_moov",
		"-",
	)

	go func() {
		stdin := cmd.Stdin()
		for {
			im, err := rd.Read()
			if err == io.EOF {
				break
			} else if err != nil {
				log.Printf("[ffmpeg] error making video: %v", err)
				break
			}
			_, err = stdin.Write(im.ToBytes())
			if err != nil {
				log.Printf("[ffmpeg] error making video: %v", err)
				break
			}
		}
		stdin.Close()
	}()

	return cmd.Stdout(), cmd
}

func Ffprobe(fname string) (width int, height int, duration float64, err error) {
	cmd := Command(
		"ffprobe", CommandOptions{NoStdin: true},
		"ffprobe",
		"-v", "error", "-select_streams", "v:0",
		"-show_entries", "stream=width,height,duration",
		"-of", "csv=s=,:p=0",
		fname,
	)
	rd := bufio.NewReader(cmd.Stdout())
	var line string
	line, err = rd.ReadString('\n')
	if err != nil {
		return
	}
	parts := strings.Split(strings.TrimSpace(line), ",")
	width, _ = strconv.Atoi(parts[0])
	height, _ = strconv.Atoi(parts[1])
	duration, _ = strconv.ParseFloat(parts[2], 64)
	cmd.Wait()
	return
}
