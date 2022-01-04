package dest_test

import (
	"github.com/swisscom/searchdump/pkg/dest"
	"strings"
	"testing"
)

func TestNewS3(t *testing.T) {
	s3dest, err := dest.NewS3(dest.S3Params{
		AccessKey:       "minio",
		SecretAccessKey: "miniostorage",
		Endpoint:        "http://127.0.0.1:9001",
		Namespace:       "ns",
		ForcePathStyle:  true,
		Region:          "us-west-1",
	})

	if err != nil {
		t.Fatalf("unable to create destS3: %v", err)
	}

	err = s3dest.Write("some-bucket",
		"test/1/1.json",
		strings.NewReader("{}"),
	)

	if err != nil {
		t.Fatalf("unable to upload: %v", err)
	}
}

