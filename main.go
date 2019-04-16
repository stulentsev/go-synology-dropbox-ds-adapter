package main

import (
	"encoding/json"
	"fmt"
	"github.com/tj/go-dropbox"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

type webhookPayload struct {
	Delta      map[string][]int    `json:"delta"`
	ListFolder map[string][]string `json:"list_folder"`
}

var (
	errlog = log.New(os.Stderr, "ERR ", log.Ldate|log.Ltime|log.Lshortfile)
	infolog = log.New(os.Stdout, "INFO ", log.Ldate|log.Ltime|log.Lshortfile)
)

func main() {
	token := os.Getenv("DROPBOX_ACCESS_TOKEN")
	if len(token) == 0 {
		log.Fatal("Failed to fetch DROPBOX_ACCESS_TOKEN from ENV")
	}
	dbx := dropbox.New(dropbox.NewConfig(token))
	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		if ch := r.URL.Query().Get("challenge"); len(ch) > 0 {
			_, _ = w.Write([]byte(ch))
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			errlog.Print(err)
			w.WriteHeader(500)
			return
		}
		jsObj := webhookPayload{}
		err = json.Unmarshal(body, &jsObj)
		if err != nil {
			errlog.Print(err)
			w.WriteHeader(500)
			return
		}

		output, err := dbx.Files.ListFolder(&dropbox.ListFolderInput{
			Path: "/Synology DownloadStation adapter",
		})
		if err != nil {
			errlog.Print(err)
			w.WriteHeader(500)
			return
		}
		for _, entry := range output.Entries {
			infolog.Printf("found new file %s\n", entry.PathLower)
			content, err := dbx.Files.Download(&dropbox.DownloadInput{
				Path: entry.PathLower,
			})
			if err != nil {
				errlog.Print(err)
				w.WriteHeader(500)
				return
			}
			infolog.Printf("Downloaded %d bytes\n", content.Length)
		}
	})

	port, ok := os.LookupEnv("PORT")
	if !ok {
		port = "3000"
	}
	infolog.Printf("Starting web server on port %s", port)
	err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
	if err != nil {
		panic(err)
	}
}
