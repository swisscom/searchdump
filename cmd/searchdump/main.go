package main

import (
	"github.com/alexflint/go-arg"
	"github.com/sirupsen/logrus"
	searchdump "github.com/swisscom/searchdump/pkg"
	"github.com/swisscom/searchdump/pkg/dest"
	"github.com/swisscom/searchdump/pkg/source"
)

var args struct {
	From     string `arg:"-f,--from,env:SEARCHDUMP_FROM,required"`
	FromType string `arg:"-F,--from-type,required"`
	To       string `arg:"-t,--to,env:SEARCHDUMP_TO,required"`
	ToType   string `arg:"-T,--to-type,required"`
	Debug    *bool  `arg:"-D,--debug"`

	// Search specific options
	SearchIndexFilter string `arg:"--search-index-filter,env:SEARCHDUMP_SEARCH_INDEX_FILTER"`
	SearchSize        int    `arg:"--search-size,env:SEARCHDUMP_SEARCH_SIZE" default:"50"`

	// S3 specific options
	S3AccessKey       string `arg:"--s3-access-key,env:SEARCHDUMP_S3_ACCESS_KEY"`
	S3SecretAccessKey string `arg:"--s3-secret-access-key,env:SEARCHDUMP_S3_SECRET_ACCESS_KEY"`
	S3Namespace       string `arg:"--s3-namespace,env:SEARCHDUMP_S3_NAMESPACE"`
	S3Endpoint        string `arg:"--s3-endpoint,env:SEARCHDUMP_S3_ENDPOINT"`
	S3ForcePathStyle  *bool  `arg:"--s3-force-path-style,env:SEARCHDUMP_S3_FORCE_PATH_STYLE"`
	S3Region          string `arg:"--s3-region,env:SEARCHDUMP_S3_REGION"`
}

func main() {
	logger := logrus.New()

	arg.MustParse(&args)
	if args.Debug != nil {
		logger.SetLevel(logrus.DebugLevel)
	}
	client := searchdump.New()
	client.SetLogger(logger)

	var err error

	switch args.FromType {
	case "search", "elasticsearch", "opensearch":
		err = client.SetFrom(args.FromType, args.From, &source.SearchParams{
		IndexFilter: args.SearchIndexFilter,
		Size: args.SearchSize,
	})
	default:
		logger.Fatalf("%s not recognized", args.FromType)
	}

	if err != nil {
		logger.Fatalf("unable to set from: %v", err)
	}

	if args.ToType == "s3" {
		forcePathStyle := false

		if args.S3ForcePathStyle != nil && *args.S3ForcePathStyle {
			forcePathStyle = true
		}

		err = client.SetTo(args.ToType, dest.S3Params{
			AccessKey:       args.S3AccessKey,
			SecretAccessKey: args.S3SecretAccessKey,
			Endpoint:        args.S3Endpoint,
			Namespace:       args.S3Namespace,
			ForcePathStyle:  forcePathStyle,
			Region:          args.S3Region,
			Url:             args.To,
		})
		if err != nil {
			logger.Fatalf("unable to setup S3: %v", err)
		}
	}

	if err != nil {
		logger.Fatalf("unable to set from: %v", err)
	}

	err = client.Start()
	if err != nil {
		logger.Fatalf("unable to run: %v", err)
	}
}
