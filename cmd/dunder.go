package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/jozuenoon/dunder/repository/cockroach"
	"github.com/jozuenoon/dunder/service"
	"github.com/jozuenoon/dunder/transport"
	"github.com/rs/zerolog"

	"github.com/gorilla/mux"
	"github.com/stevenroose/gonfig"
	"gopkg.in/go-playground/validator.v9"
)

var config = struct {
	TLS      bool   `id:"tls" desc:"Connection uses TLS if true, else plain TCP" validate:"required"`
	CertFile string `id:"cert_file" desc:"TLS certificate file path" validate:"required"`
	KeyFile  string `id:"key_file" desc:"TLS key file path" validate:"required"`
	Port     int    `id:"port" desc:"GRPC port" validate:"required"`

	LogLevel string `id:"log_level" desc:"Options: debug, info, warn, error, fatal, panic"`

	CockroachDB *CockroachDBConfig `id:"cockroach"`

	ConfigFile string `id:"config_file" desc:"provide a config file path"`
}{
	Port: 9000,
	LogLevel: "debug",
}

//go:generate gomodifytags -file dunder.go -struct CockroachDBConfig -add-tags id -w
type CockroachDBConfig struct {
	Host          string `id:"host"`
	ShouldMigrate bool   `id:"should_migrate"`
	Debug         bool   `id:"debug"`
	Database      string `id:"database"`
	User          string `id:"user"`
}

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	level, err := zerolog.ParseLevel(config.LogLevel)
	if err != nil {
		panic(err)
	}
	zerolog.SetGlobalLevel(level)
	log := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()

	if err := gonfig.Load(&config, gonfig.Conf{
		ConfigFileVariable:  "config_file",
		FileDefaultFilename: "/config/config.yaml",
		FileDecoder:         gonfig.DecoderYAML,
	}); err != nil {
		log.Fatal().Err(err).Msg("could not load config")
	}

	if err := validator.New().Struct(config); err != nil {
		log.Fatal().Err(err).Msg("config validation failed")
	}

	repoSvc, err := cockroach.New(&cockroach.Config{
		Host:          config.CockroachDB.Host,
		ShouldMigrate: config.CockroachDB.ShouldMigrate,
		Debug:         config.CockroachDB.Debug,
		Database:      &config.CockroachDB.Database,
		User:          &config.CockroachDB.User,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create cockroach repo")
	}

	dunder := service.NewDunder(repoSvc, &log)
	dunderSearch := service.NewDunderSearch(repoSvc, &log)

	dunderHttp := transport.NewHttp(dunder, dunderSearch, &log)

	r := mux.NewRouter()
	r.HandleFunc("/message", dunderHttp.CreateMessage).Methods(http.MethodPost)
	r.HandleFunc("/message", dunderHttp.MessageQuery).Methods(http.MethodGet)
	r.HandleFunc("/message/{ulid}", dunderHttp.MessageQuery).Methods(http.MethodGet)
	r.HandleFunc("/trend", dunderHttp.Trends).Methods(http.MethodGet)

	if config.TLS {
		if err := http.ListenAndServeTLS(fmt.Sprintf(":%d", config.Port), config.CertFile, config.KeyFile, r); err != nil {
			log.Fatal().Err(err).Msg("server failed")
		}
	}
	if err := http.ListenAndServe(fmt.Sprintf(":%d", config.Port), r); err != nil {
		log.Fatal().Err(err).Msg("server failed")
	}
}
