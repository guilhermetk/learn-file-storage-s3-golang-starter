// Package util contains utils files for the project
package util

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"os/exec"
	"strings"
)

type ffprobeOutput struct {
	Streams []struct {
		CodecType string `json:"codec_type"`
		Width     int    `json:"width,omitempty"`
		Height    int    `json:"height,omitempty"`
	} `json:"streams"`
}

func GetVideoAspectRatio(filePath string) (string, error) {
	ffprobeArgs := []string{"-v", "error", "-print_format", "json", "-show_streams", filePath}
	cmd := exec.Command("ffprobe", ffprobeArgs...)
	output := new(bytes.Buffer)
	cmd.Stdout = output
	err := cmd.Run()
	if err != nil {
		return "", errors.New("unable to run ffprobe command")
	}

	var outputJSON ffprobeOutput
	err = json.Unmarshal(output.Bytes(), &outputJSON)
	if err != nil {
		return "", errors.New("unable to parse ffprobe output into json")
	}

	for _, stream := range outputJSON.Streams {
		if stream.CodecType == "video" {
			width, height := stream.Width, stream.Height

			if width == 0 || height == 0 {
				return "", errors.New("invalid dimensions")
			}

			aspectRatio := math.Round(float64(width) / float64(height))

			if aspectRatio == math.Round(9.0/16.0) {
				return "9:16", nil
			}

			if aspectRatio == math.Round(16.0/9.0) {
				return "16:9", nil
			}

			return "other", nil
		}
	}

	return "", nil
}

func ProcessVideoForFastStart(filePath string) (string, error) {
	processingFileName := strings.Join([]string{filePath, "processing"}, ".")

	cmdArgs := []string{"-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", processingFileName}
	cmd := exec.Command("ffmpeg", cmdArgs...)

	err := cmd.Run()
	if err != nil {
		return "", errors.New("unable to process video for faststart")
	}

	return processingFileName, nil
}
