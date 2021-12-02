// Code generated by Wire. DO NOT EDIT.

//go:generate wire
//go:build !wireinject
// +build !wireinject

package operation

import (
	db2 "github.com/aquasecurity/trivy-db/pkg/db"
	"github.com/aquasecurity/trivy/pkg/db"
	"github.com/aquasecurity/trivy/pkg/github"
	"github.com/aquasecurity/trivy/pkg/indicator"
	"github.com/spf13/afero"
	"k8s.io/utils/clock"
)

// Injectors from inject.go:

func initializeDBClient(cacheDir string, quiet bool) db.Client {
	config := db2.Config{}
	client := github.NewClient()
	progressBar := indicator.NewProgressBar(quiet)
	realClock := clock.RealClock{}
	fs := afero.NewOsFs()
	metadata := db.NewMetadata(fs, cacheDir)
	dbClient := db.NewClient(config, client, progressBar, realClock, metadata)
	return dbClient
}
