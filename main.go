package main

import (
	"encoding/json"
	"fmt"
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
	errlog  = log.New(os.Stderr, "ERR ", log.Ldate|log.Ltime|log.Lshortfile)
	infolog = log.New(os.Stdout, "INFO ", log.Ldate|log.Ltime|log.Lshortfile)

	dropboxToken = os.Getenv("DROPBOX_ACCESS_TOKEN")
)

func main() {
	if len(dropboxToken) == 0 {
		log.Fatal("Failed to fetch DROPBOX_ACCESS_TOKEN from ENV")
	}

	outputFolder := os.Getenv("OUTPUT_FOLDER")
	if len(outputFolder) == 0 {
		log.Fatal("Failed to fetch OUTPUT_FOLDER from ENV")
	}

	in := startPipeline(
		listFolder(),
		filterFileTypes(".torrent", ".pdf"),
		stopSeenEntries(),
		downloadToFolder(outputFolder),
		markAsProcessed(),
	)
	defer close(in) // probably redundant

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

		for _, acc := range jsObj.ListFolder["accounts"] {
			infolog.Printf("detected change for account %s", acc)
			in <- acc
		}
		w.WriteHeader(http.StatusOK)
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
