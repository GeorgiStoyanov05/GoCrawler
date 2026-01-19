package main

import (
	"html/template"
	"log"
	"net/http"

	"GoCrawler/internal/storage"
)

type TemplateData struct {
	Results []ImageResult
}

type ImageResult struct {
	Thumbnail string
	FullImage string
	Filename  string
	Format    string
	URL       string
}

func main() {
	store, err := storage.NewMySQLStorage(
		"crawler",
		"password123",
		"localhost:3306",
		"crawlerdb",
	)
	if err != nil {
		log.Fatal("DB error:", err)
	}
	repo := storage.NewImageRepository(store)

	tmpl, err := template.ParseFiles("internal/web/templates/index.html")
	if err != nil {
		log.Fatal("template parse error:", err)
	}

	http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir("./images"))))
	http.Handle("/thumbnails/", http.StripPrefix("/thumbnails/", http.FileServer(http.Dir("./thumbnails"))))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		params := map[string]string{
			"format":   r.URL.Query().Get("format"),
			"filename": r.URL.Query().Get("filename"),
			"url":      r.URL.Query().Get("url"),
		}

		results, err := repo.SearchImages(r.Context(), params)
		if err != nil {
			http.Error(w, "DB search error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		imgs := make([]ImageResult, 0, len(results))
		for _, m := range results {
			imgs = append(imgs, ImageResult{
				Thumbnail: m.ThumbPath,
				FullImage: m.SavedPath,
				Filename:  m.Filename,
				Format:    m.Format,
				URL:       m.OriginalURL,
			})
		}

		if err := tmpl.Execute(w, TemplateData{Results: imgs}); err != nil {
			http.Error(w, "template execute error: "+err.Error(), http.StatusInternalServerError)
			return
		}
	})

	log.Println("Web server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
