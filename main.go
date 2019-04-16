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
			fmt.Println("ERR:", err)
			w.WriteHeader(500)
			return
		}
		jsObj := webhookPayload{}
		err = json.Unmarshal(body, &jsObj)
		if err != nil {
			fmt.Println("ERR:", err)
			w.WriteHeader(500)
			return
		}

		fmt.Println(jsObj)
		output, err := dbx.Files.ListFolder(&dropbox.ListFolderInput{
			Path: "/Synology DownloadStation adapter",
		})
		if err != nil {
			fmt.Println("ERR:", err)
			w.WriteHeader(500)
			return
		}
		for _, entry := range output.Entries {
			fmt.Printf("found file %s\n", entry.PathLower)
			content, err := dbx.Files.Download(&dropbox.DownloadInput{
				Path: entry.PathLower,
			})
			if err != nil {
				fmt.Println("ERR:", err)
				w.WriteHeader(500)
				return
			}
			fmt.Printf("Downloaded %d bytes\n", content.Length)
		}
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, "Hello, you've requested: %s\n", r.URL.Path)
	})

	port, ok := os.LookupEnv("PORT")
	if !ok {
		port = "3000"
	}
	err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
	if err != nil {
		panic(err)
	}
}
