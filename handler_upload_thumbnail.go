package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
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

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	const maxThumbnailSize = 10 << 20

	err = r.ParseMultipartForm(maxThumbnailSize)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse form", err)
		return
	}

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Coudn't extract thumbnail data from form", err)
		return
	}
	defer file.Close()

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't find video with given id", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", err)
		return
	}

	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Content-Type", err)
		return
	}
	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusBadRequest, "Invalid file type", nil)
		return
	}

	fileExtension := strings.Split(mediaType, "/")[1]
	rbS := make([]byte, 32)
	_, err = rand.Read(rbS)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal error", nil)
		return
	}
	randomFileName := base64.RawURLEncoding.EncodeToString(rbS)
	fileNameWithExtension := strings.Join([]string{randomFileName, fileExtension}, ".")
	filePath := filepath.Join("assets", fileNameWithExtension)

	newFile, err := os.Create(filePath)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't create new thumbnail file", err)
		return
	}

	if _, err := io.Copy(newFile, file); err != nil {
		respondWithError(w, http.StatusBadRequest, "Error saving thumbnail file to disk", err)
		return
	}

	// http://localhost:<port>/assets/<videoID>.<file_extension>
	thumbnailURL := fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, fileNameWithExtension)
	video.ThumbnailURL = &thumbnailURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to update video metadata", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
