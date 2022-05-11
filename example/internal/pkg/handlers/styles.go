package handlers

import (
	"embed"
	"fmt"
	"log"
	"net/http"
)

//go:embed css/*
var cssContent embed.FS

func StylesHandler(w http.ResponseWriter, r *http.Request) {
	normalizeData, err := cssContent.ReadFile("css/normalize.min.css")
	if err != nil {
		err = fmt.Errorf("failed to load normalize css content from storage: %w", err)
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	tachyonsData, err := cssContent.ReadFile("css/tachyons.min.css")
	if err != nil {
		err = fmt.Errorf("failed to load tachyons css content from storage: %w", err)
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	siteStyleData, err := cssContent.ReadFile("css/styles.css")
	if err != nil {
		err = fmt.Errorf("failed to load app styles css content from storage: %w", err)
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	allCSSData := ""
	for _, b := range []*[]byte{&normalizeData, &tachyonsData, &siteStyleData} {
		allCSSData += string(*b) + "\n"
	}

	w.Header().Set("Content-Type", "text/css")
	fmt.Fprint(w, allCSSData)
}
