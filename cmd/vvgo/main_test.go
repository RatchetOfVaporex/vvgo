package main

import (
	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/assert"
	"github.com/virtual-vgo/vvgo/pkg/api"
	"github.com/virtual-vgo/vvgo/pkg/storage"
	"github.com/virtual-vgo/vvgo/pkg/tracing"
	"os"
	"testing"
)

func TestConfig_ParseEnv(t *testing.T) {
	envs := map[string]string{
		"VVGO_SECRET":                    "vvgo-secret",
		"INITIALIZE_STORAGE":             "true",
		"TRACING_HONEYCOMB_DATASET":      "tracing-honeycomb-dataset",
		"TRACING_HONEYCOMB_WRITE_KEY":    "tracing-honeycomb-write-key",
		"TRACING_SERVICE_NAME":           "tracing-service-name",
		"API_LISTEN_ADDRESS":             "listen-address",
		"API_MAX_CONTENT_LENGTH":         "1000000",
		"API_STORAGE_SHEETS_BUCKET_NAME": "sheets-bucket-name",
		"API_STORAGE_CLIX_BUCKET_NAME":   "clix-bucket-name",
		"API_STORAGE_TRACKS_BUCKET_NAME": "tracks-bucket-name",
		"API_STORAGE_PARTS_HASH_KEY":     "parts-hash-key",
		"API_STORAGE_PARTS_LOCKER_KEY":   "parts-locker-key",
		"API_MEMBER_USER":                "member-user",
		"API_MEMBER_PASS":                "member-pass",
		"API_PREP_REP_TOKEN":             "prep-rep-token",
		"API_ADMIN_TOKEN":                "admin-token",
		"STORAGE_MINIO_ENDPOINT":         "minio-endpoint",
		"STORAGE_MINIO_REGION":           "minio-region",
		"STORAGE_MINIO_ACCESSKEY":        "minio-access-key",
		"STORAGE_MINIO_SECRETKEY":        "minio-secret-key",
		"STORAGE_MINIO_USESSL":           "true",
	}
	want := Config{
		Secret:            "vvgo-secret",
		InitializeStorage: true,
		TracingConfig: tracing.Config{
			HoneycombWriteKey: "tracing-honeycomb-write-key",
			HoneycombDataset:  "tracing-honeycomb-dataset",
			ServiceName:       "tracing-service-name",
		},
		ApiConfig: api.ServerConfig{
			ListenAddress:    "listen-address",
			MaxContentLength: 1e6,
			AdminToken:       "admin-token",
			PrepRepToken:     "prep-rep-token",
			MemberUser:       "member-user",
			MemberPass:       "member-pass",
		},
		ApiStorageConfig: api.StorageConfig{
			SheetsBucketName: "sheets-bucket-name",
			ClixBucketName:   "clix-bucket-name",
			TracksBucketName: "tracks-bucket-name",
			PartsHashKey:     "parts-hash-key",
			PartsLockerKey:   "parts-locker-key",
		},
		StorageConfig: storage.Config{
			Minio: storage.MinioConfig{
				Endpoint:  "minio-endpoint",
				Region:    "minio-region",
				AccessKey: "minio-access-key",
				SecretKey: "minio-secret-key",
				UseSSL:    true,
			},
		},
	}

	for k, v := range envs {
		os.Setenv(k, v)
	}
	var got Config
	envconfig.Usage("", &got)
	got.ParseEnv()
	assert.Equal(t, want, got)
}
