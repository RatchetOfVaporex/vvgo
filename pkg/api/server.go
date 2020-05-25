package api

import (
	"context"
	"github.com/virtual-vgo/vvgo/pkg/log"
	"github.com/virtual-vgo/vvgo/pkg/login"
	"github.com/virtual-vgo/vvgo/pkg/parts"
	"github.com/virtual-vgo/vvgo/pkg/storage"
	"github.com/virtual-vgo/vvgo/pkg/tracing"
	"net/http"
	"net/http/pprof"
)

var logger = log.Logger()

var PublicFiles = "public"

type ServerConfig struct {
	ListenAddress     string       `split_words:"true" default:"0.0.0.0:8080"`
	MaxContentLength  int64        `split_words:"true" default:"10000000"`
	MemberUser        string       `split_words:"true" default:"admin"`
	MemberPass        string       `split_words:"true" default:"admin"`
	UploaderToken     string       `split_words:"true" default:"admin"`
	DeveloperToken    string       `split_words:"true" default:"admin"`
	DistroBucketName  string       `split_words:"true" default:"vvgo-distro"`
	BackupsBucketName string       `split_words:"true" default:"backups"`
	RedisNamespace    string       `split_words:"true" default:"local"`
	Login             login.Config `envconfig:"login"`
}

func NewServer(ctx context.Context, config ServerConfig) *http.Server {
	var newBucket = func(ctx context.Context, bucketName string) *storage.Bucket {
		bucket, err := storage.NewBucket(ctx, bucketName)
		if err != nil {
			logger.WithError(err).WithField("bucket_name", bucketName).Fatal("storage.NewBucket() failed")
		}
		return bucket
	}

	database := Database{
		Distro:   newBucket(ctx, config.DistroBucketName),
		Parts:    parts.NewParts(config.RedisNamespace),
		Sessions: login.NewStore(config.RedisNamespace, config.Login),
	}

	mux := RBACMux{
		Basic: map[[2]string][]login.Role{
			{config.MemberUser, config.MemberPass}:    {login.RoleVVGOMember},
			{"vvgo-uploader", config.UploaderToken}:   {login.RoleVVGOUploader, login.RoleVVGOMember},
			{"vvgo-developer", config.DeveloperToken}: {login.RoleVVGODeveloper, login.RoleVVGOUploader, login.RoleVVGOMember},
		},
		Bearer: map[string][]login.Role{
			config.UploaderToken:  {login.RoleVVGOUploader, login.RoleVVGOMember},
			config.DeveloperToken: {login.RoleVVGODeveloper, login.RoleVVGOUploader, login.RoleVVGOMember},
		},
		ServeMux: http.NewServeMux(),
		Sessions: database.Sessions,
	}

	mux.Handle("/auth", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("authenticated"))
	}), login.RoleVVGOUploader)

	// debug endpoints from net/http/pprof
	mux.HandleFunc("/debug/pprof/", pprof.Index, login.RoleVVGODeveloper)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline, login.RoleVVGODeveloper)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile, login.RoleVVGODeveloper)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol, login.RoleVVGODeveloper)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace, login.RoleVVGODeveloper)

	mux.Handle("/parts", PartView{Database: &database}, login.RoleVVGOMember)

	backups := newBucket(ctx, config.BackupsBucketName)
	mux.Handle("/backups", &BackupHandler{
		Database: &database,
		Backups:  backups,
	}, login.RoleVVGODeveloper)

	mux.Handle("/download", DownloadHandler{
		config.DistroBucketName:  database.Distro.DownloadURL,
		config.BackupsBucketName: backups.DownloadURL,
	}, login.RoleVVGOMember)

	// Uploads
	mux.Handle("/upload", UploadHandler{
		Database: &database,
	}, login.RoleVVGOUploader)

	// Projects
	mux.Handle("/projects", ProjectsHandler{}, login.RoleVVGOUploader)

	mux.Handle("/version", http.HandlerFunc(Version), login.RoleAnonymous)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			IndexView{}.ServeHTTP(w, r)
		} else {
			http.FileServer(http.Dir("public")).ServeHTTP(w, r)
		}
	}, login.RoleAnonymous)

	return &http.Server{
		Addr:     config.ListenAddress,
		Handler:  tracing.WrapHandler(&mux),
		ErrorLog: log.StdLogger(),
	}
}
