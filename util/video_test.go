package util

import "testing"

func TestGetVideoAspectRatio(t *testing.T) {
	t.Run("Landscape aspect ration", func(t *testing.T) {
		expected := "16:9"
		got, err := GetVideoAspectRatio("/home/gtiscoski/personal/code/learn-file-storage-s3-golang-starter/samples/boots-video-horizontal.mp4")
		if err != nil {
			t.Error(err)
		}

		if expected != got {
			t.Errorf("Invalid video aspect ratio Expected: %s Got: %s", expected, got)
		}
	})

	t.Run("Portrait aspect ration", func(t *testing.T) {
		expected := "9:16"
		got, err := GetVideoAspectRatio("/home/gtiscoski/personal/code/learn-file-storage-s3-golang-starter/samples/boots-video-vertical.mp4")
		if err != nil {
			t.Error(err)
		}

		if expected != got {
			t.Errorf("Invalid video aspect ratio Expected: %s Got: %s", expected, got)
		}
	})
}
