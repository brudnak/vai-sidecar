package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	_ "modernc.org/sqlite"
)

func main() {
	log.Println("VAI Sidecar starting on :8080")

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	http.HandleFunc("/snapshot", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Snapshot requested")

		dbPath := "/var/lib/rancher/informer_object_cache.db"
		snapshotPath := fmt.Sprintf("/tmp/snapshot-%d.db", time.Now().Unix())

		db, err := sql.Open("sqlite", dbPath+"?mode=ro")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer db.Close()

		_, err = db.Exec(fmt.Sprintf("VACUUM INTO '%s'", snapshotPath))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		defer os.Remove(snapshotPath)

		file, err := os.Open(snapshotPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer file.Close()

		w.Header().Set("Content-Type", "application/octet-stream")
		io.Copy(w, file)
		log.Println("Snapshot sent successfully")
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
