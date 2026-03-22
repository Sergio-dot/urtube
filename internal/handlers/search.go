package handlers

import (
	"fmt"
	"net/http"

	"github.com/Sergio-dot/urtube/pkg"
	"github.com/lrstanley/go-ytdlp"
)

func SearchVideo(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	param := r.PathValue("searchParam")
	if pkg.IsEmpty(param) {
		return APIError{StatusCode: http.StatusBadRequest, Message: "search parameter is required"}
	}

	searchStr := fmt.Sprintf("ytsearch5:%s", param)

	command, err := ytdlp.New().
		PrintJSON().
		SkipDownload().
		FlatPlaylist().
		Run(ctx, searchStr)
	if err != nil {
		return fmt.Errorf("failed to run ytdlp: %w", err)
	}

	searchResults, err := command.GetExtractedInfo()
	if err != nil {
		return fmt.Errorf("failed to extract video info: %w", err)
	}
	if len(searchResults) == 0 {
		return APIError{StatusCode: http.StatusNotFound, Message: "no results found"}
	}

	JSON(w, http.StatusOK, searchResults)
	return nil
}
