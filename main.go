// VAI snapshot side-car
//
// Endpoints
//
//	GET /health          →  200 OK
//	GET /snapshot        →  application/octet-stream (download)
//	GET /snapshot/s3     →  {"bucket":"…","key":"…","url":"s3://…"}  (upload)
//
// Required ENV
//
//	SNAPSHOT_BUCKET      –  S3 bucket name  (e.g. "vai-snapshots")
//	POD_NAME             –  injected via fieldRef metadata.name
//
// Optional ENV
//
//	SNAPSHOT_PREFIX      –  "folder/" inside the bucket (default "")
//	AWS_*                –  credentials / region, OR use IRSA / workload identity
//
// Build: CGO_ENABLED=0 go build -o vai-sidecar .
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	_ "modernc.org/sqlite"
)

const (
	dbPath    = "/var/lib/rancher/informer_object_cache.db"
	ListenOn  = ":8080"
	timeStamp = "20060102-150405"
)

// ---------- env helpers ----------------------------------------------------

var (
	bucket = mustEnv("SNAPSHOT_BUCKET")   // e.g. vai-snapshots
	prefix = os.Getenv("SNAPSHOT_PREFIX") // "" or "dev/"
	pod    = mustEnv("POD_NAME")          // injected by fieldRef
)

// crash if missing
func mustEnv(k string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	log.Fatalf("env %s is required", k)
	return ""
}

// ---------- S3 uploader singleton ------------------------------------------

func newUploader(ctx context.Context) *manager.Uploader {
	cfg, err := config.LoadDefaultConfig(ctx) // region picked from env / creds
	if err != nil {
		log.Fatalf("AWS cfg: %v", err)
	}
	return manager.NewUploader(s3.NewFromConfig(cfg))
}

// ---------- snapshot logic (file + optional upload) ------------------------

func buildSnapshot() (tmpPath string, err error) {
	tmpPath = fmt.Sprintf("/tmp/vai-%s.db", time.Now().Format(timeStamp))

	db, err := sql.Open("sqlite", dbPath+"?mode=ro")
	if err != nil {
		return "", err
	}
	if _, err = db.Exec(fmt.Sprintf(`VACUUM INTO '%s'`, tmpPath)); err != nil {
		return "", err
	}
	db.Close()
	return tmpPath, nil
}

func uploadToS3(ctx context.Context, up *manager.Uploader, filePath string) (key string, err error) {
	key = fmt.Sprintf("%s%s-%s.db", prefix, pod, time.Now().Format(timeStamp))
	f, _ := os.Open(filePath)
	defer f.Close()

	_, err = up.Upload(ctx, &s3.PutObjectInput{
		Bucket: &bucket,
		Key:    &key,
		Body:   f,
	})
	return key, err
}

// ---------- HTTP handlers --------------------------------------------------

func main() {
	log.Printf("VAI side-car on %s, bucket=%s\n", ListenOn, bucket)
	ctx := context.Background()
	uploader := newUploader(ctx)

	http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// --- download endpoint --------------------------------------------------
	http.HandleFunc("/snapshot", func(w http.ResponseWriter, r *http.Request) {
		log.Println("[INFO] /snapshot requested")
		tmp, err := buildSnapshot()
		if err != nil {
			log.Println("[ERR] snapshot:", err)
			http.Error(w, err.Error(), 500)
			return
		}
		defer os.Remove(tmp)

		w.Header().Set("Content-Type", "application/octet-stream")
		f, _ := os.Open(tmp)
		defer f.Close()
		_, _ = io.Copy(w, f)
	})

	// --- S3 upload endpoint -------------------------------------------------
	http.HandleFunc("/snapshot/s3", func(w http.ResponseWriter, r *http.Request) {
		log.Println("[INFO] /snapshot/s3 requested")
		tmp, err := buildSnapshot()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer os.Remove(tmp)

		key, err := uploadToS3(r.Context(), uploader, tmp)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		resp := map[string]string{
			"bucket": bucket,
			"key":    key,
			"url":    fmt.Sprintf("s3://%s/%s", bucket, key),
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	log.Fatal(http.ListenAndServe(ListenOn, nil))
}
