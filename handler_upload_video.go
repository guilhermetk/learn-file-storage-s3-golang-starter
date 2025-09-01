package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/util"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	const maxUploadSize = 1 << 30
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't find video with given id", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Coudn't extract video data from form", err)
		return
	}

	defer file.Close()

	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Content-Type", err)
		return
	}
	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Invalid file type", nil)
		return
	}

	tmpFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Invalid file type", nil)
		return
	}

	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	io.Copy(tmpFile, file)
	tmpFile.Seek(0, io.SeekStart)

	fastStartFilePath, err := util.ProcessVideoForFastStart(tmpFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to process video for faststart", nil)
		return
	}

	fastStartFile, err := os.Open(fastStartFilePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to open processed video for faststart", nil)
		return
	}
	defer os.Remove(fastStartFile.Name())
	defer fastStartFile.Close()

	aspectRatio, err := util.GetVideoAspectRatio(fastStartFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to get video aspect ratio", nil)
		return
	}
	prefix := prefixFromAspectRatio(aspectRatio)

	fileExtension := strings.Split(mediaType, "/")[1]
	rbS := make([]byte, 32)
	_, err = rand.Read(rbS)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal error", nil)
		return
	}
	randomFileName := base64.RawURLEncoding.EncodeToString(rbS)
	fileNameWithExtension := strings.Join([]string{randomFileName, fileExtension}, ".")
	fileNameWithPrefixExtension := strings.Join([]string{prefix, fileNameWithExtension}, "/")

	params := s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &fileNameWithPrefixExtension,
		Body:        fastStartFile,
		ContentType: &mediaType,
	}
	_, err = cfg.s3Client.PutObject(context.TODO(), &params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to upload video file to S3", nil)
		return
	}

	videoURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, fileNameWithPrefixExtension)
	video.VideoURL = &videoURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to update video metadata", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}

func prefixFromAspectRatio(a string) string {
	if a == "16:9" {
		return "landscape"
	}

	if a == "9:16" {
		return "portrait"
	}

	return "other"
}
