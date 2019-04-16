package main

import (
	"github.com/tj/go-dropbox"
	"io"
	"os"
	"path/filepath"
)

type pipelineSegment interface {
	Process(in, out chan string)
}

type pipelineSegmentFunc func(in, out chan string)

func (f pipelineSegmentFunc) Process(in, out chan string) {
	f(in, out)
}

func startPipeline(segments ...pipelineSegment) chan string {
	var in, out chan string
	entryCh := make(chan string, 10)
	in = entryCh
	out = make(chan string)

	for _, thisSegment := range segments {
		go func(segment pipelineSegment, in, out chan string) {
			segment.Process(in, out) // will block until `in` is closed
			close(out)
		}(thisSegment, in, out)

		in = out
		out = make(chan string)
	}

	return entryCh
}

// starts or continues folder scan
// TODO: make it multi-user capable
func listFolder() pipelineSegment {
	appFolder := "/Synology DownloadStation adapter" // TODO: probably need to make this configurable
	dbx := dropbox.New(dropbox.NewConfig(dropboxToken))
	var cursor string

	return pipelineSegmentFunc(func(in, out chan string) {
		infolog.Print("Starting listFolder segment")
		defer infolog.Print("Stopping listFolder segment")
		for acc := range in {
			infolog.Printf("listing directory for account %s", acc)
			var output *dropbox.ListFolderOutput
			var err error
			if len(cursor) == 0 {
				output, err = dbx.Files.ListFolder(&dropbox.ListFolderInput{
					Path:           appFolder,
					IncludeDeleted: false,
				})
			} else {
				output, err = dbx.Files.ListFolderContinue(&dropbox.ListFolderContinueInput{
					Cursor: cursor,
				})
			}

			if err != nil {
				errlog.Print(err)
				return
			}
			cursor = output.Cursor

			for _, entry := range output.Entries {
				infolog.Printf("seeing file %s", entry.PathLower)
				out <- entry.PathLower
			}
		}
	})
}

func stopSeenEntries() pipelineSegment {

	seenEntries := make(map[string]struct{})

	return pipelineSegmentFunc(func(in, out chan string) {
		infolog.Print("Starting stopSeenEntries segment")
		defer infolog.Print("Stopping stopSeenEntries segment")
		for filename := range in {
			_, ok := seenEntries[filename]
			if !ok {
				infolog.Printf("previously unseen file %s, passing for downloading", filename)
				seenEntries[filename] = struct{}{}
				out <- filename
			}
		}
	})
}

func filterFileTypes(fileTypes ...string) pipelineSegment {
	return pipelineSegmentFunc(func(in, out chan string) {
		infolog.Print("Starting filterFileTypes segment")
		defer infolog.Print("Stopping filterFileTypes segment")
		for filename := range in {
			realExt := filepath.Ext(filename)

			for _, ext := range fileTypes {
				if ext == realExt {
					out <- filename
					continue
				}
			}
		}
	})
}

func downloadToFolder(outputFolder string) pipelineSegment {
	dbx := dropbox.New(dropbox.NewConfig(dropboxToken))

	return pipelineSegmentFunc(func(in, out chan string) {
		infolog.Print("Starting downloadToFolder segment")
		defer infolog.Print("Stopping downloadToFolder segment")
		for filename := range in {
			outputFilename := filepath.Join(
				outputFolder,
				filepath.Base(filename),
			)
			infolog.Printf("downloading %s -> %s", filename, outputFilename)

			content, err := dbx.Files.Download(&dropbox.DownloadInput{
				Path: filename,
			})
			if err != nil {
				errlog.Print(err)
				return
			}

			file, err := os.Create(outputFilename)
			if err != nil {
				errlog.Printf("can't open file %s for writing: %v", outputFilename, err)
			}

			n, err := io.Copy(file, content.Body)
			if err != nil {
				errlog.Print(err)
			}
			infolog.Printf("downloaded %d bytes", n)
			out <- filename

			_ = file.Close()
			_ = content.Body.Close()
		}
	})
}

func markAsProcessed() pipelineSegment {
	dbx := dropbox.New(dropbox.NewConfig(dropboxToken))

	return pipelineSegmentFunc(func(in, out chan string) {
		for filename := range in {
			_, err := dbx.Files.Move(&dropbox.MoveInput{
				FromPath: filename,
				ToPath:   filename + ".processed",
			})

			if err != nil {
				errlog.Print(err)
			}
		}
	})
}
