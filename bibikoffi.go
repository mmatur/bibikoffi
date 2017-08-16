package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/BurntSushi/toml"
	"github.com/containous/bibikoffi/internal/gh"
	"github.com/containous/bibikoffi/mjolnir"
	"github.com/containous/bibikoffi/types"
	"github.com/containous/flaeg"
)

func main() {

	options := &types.Options{
		DryRun:         true,
		Debug:          false,
		ConfigFilePath: "./bibikoffi.toml",
	}

	defaultPointersOptions := &types.Options{}
	rootCmd := &flaeg.Command{
		Name:                  "bibikoffi",
		Description:           `Myrmica Bibikoffi: Closes stale issues.`,
		Config:                options,
		DefaultPointersConfig: defaultPointersOptions,
		Run: func() error {
			if options.Debug {
				log.Printf("Run bibikoffi command with config : %+v\n", options)
			}

			if len(options.GitHubToken) == 0 {
				options.GitHubToken = os.Getenv("GITHUB_TOKEN")
			}

			required(options.GitHubToken, "token")
			required(options.ConfigFilePath, "config-path")

			if options.DryRun {
				log.Print("IMPORTANT: you are using the dry-run mode. Use `--dry-run=false` to disable this mode.")
			}

			err := process(options)
			if err != nil {
				log.Fatal(err)
			}
			return nil
		},
	}

	flag := flaeg.New(rootCmd, os.Args[1:])
	flag.Run()
}

func process(options *types.Options) error {
	if options.ServerMode {
		server := &server{options: options}
		return server.ListenAndServe()
	}

	return launch(options)
}

func launch(options *types.Options) error {

	config := &types.Configuration{}
	meta, err := toml.DecodeFile(options.ConfigFilePath, config)

	if err != nil {
		return err
	}

	if options.Debug {
		log.Printf("configuration: %+v\n", meta)
	}

	ctx := context.Background()
	client := gh.NewGitHubClient(ctx, options.GitHubToken)

	return mjolnir.CloseIssues(client, ctx, config.Owner, config.RepositoryName, config.Rules, options.DryRun, options.Debug)
}

func required(field string, fieldName string) error {
	if len(field) == 0 {
		log.Fatalf("%s is mandatory.", fieldName)
	}
	return nil
}

type server struct {
	options *types.Options
}

func (s *server) ListenAndServe() error {
	return http.ListenAndServe(":"+strconv.Itoa(s.options.ServerPort), s)
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.Printf("Invalid http method: %s", r.Method)
		http.Error(w, "405 Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := launch(s.options)
	if err != nil {
		log.Printf("Report error: %v", err)
		http.Error(w, "Report error.", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, "Scheluded.")
}
