package cli

import (
	"embed"
	"fmt"
	"html/template"
	"khoai-chain/internal/config"
	"net/http"
)

// Khai bao he thong tep ao chua toan bo file trong thu muc templates
//
//go:embed templates/*
var uiFiles embed.FS

func RunUI(targetNode string) error {
	// 1. Chi parse file HTML de lam template truyen bien
	tmpl, err := template.ParseFS(uiFiles, "templates/index.html")
	if err != nil {
		return fmt.Errorf("error parse template: %w", err)
	}

	mux := http.NewServeMux()

	// 2. Dinh tuyen rieng cho file CSS
	mux.HandleFunc("/style.css", func(w http.ResponseWriter, r *http.Request) {
		data, _ := uiFiles.ReadFile("templates/style.css")
		w.Header().Set("Content-Type", "text/css")
		w.Write(data)
	})

	// 3. Dinh tuyen rieng cho file JS
	mux.HandleFunc("/script.js", func(w http.ResponseWriter, r *http.Request) {
		data, _ := uiFiles.ReadFile("templates/script.js")
		w.Header().Set("Content-Type", "application/javascript")
		w.Write(data)
	})

	// 4. Dinh tuyen cho HTML va tra ve loi 404 neu go sai duong dan
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		data := struct {
			TargetNode string
		}{
			TargetNode: targetNode,
		}

		// Render template co chua bien, KHONG dung w.Write nua
		tmpl.Execute(w, data)
	})

	fmt.Printf("The UI is currently running at http://localhost%s (Connect to Node: %s)\n", config.Endpoint, targetNode)

	return http.ListenAndServe(config.Endpoint, mux)
}
