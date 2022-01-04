package dest

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/swisscom/searchdump/pkg/file"
	"net/url"
	"path"
)

type S3Params struct {
	AccessKey       string
	Endpoint        string
	Namespace       string
	ForcePathStyle  bool
	SecretAccessKey string
	Region          string
	Url             string
}

type S3Dest struct {
	c        *s3.Client
	bucket   string
	path     string
	params   *S3Params
	uploader *manager.Uploader
}

func (d *S3Dest) String() string {
	return fmt.Sprintf("S3 (endpoint=%s, accessKey=%s, region=%s)",
		d.params.Endpoint,
		d.params.AccessKey,
		d.params.Region,
	)
}

var _ Dester = (*S3Dest)(nil)

func (d *S3Dest) Write(file file.File) error {
	if d.uploader == nil {
		return fmt.Errorf("unable to write: uploader is nil")
	}

	key := path.Join(d.path, file.Name)

	_, err := d.uploader.Upload(context.Background(), &s3.PutObjectInput{
		Bucket: &d.bucket,
		Key:    &key,
		Body:   file.Content,
	})
	return err

}

type EndpointResolver struct {
	endpointUrl string
}

var _ aws.EndpointResolverWithOptions = (*EndpointResolver)(nil)

func (e EndpointResolver) ResolveEndpoint(service, region string, options ...interface{}) (aws.Endpoint, error) {
	endpoint := aws.Endpoint{
		URL:               e.endpointUrl,
		HostnameImmutable: true,
		Source:            aws.EndpointSourceCustom}
	return endpoint, nil
}

func NewS3(params S3Params) (*S3Dest, error) {
	dest := S3Dest{}
	credsProvider := credentials.NewStaticCredentialsProvider(
		params.AccessKey,
		params.SecretAccessKey,
		"",
	)

	u, err := url.Parse(params.Url)
	if err != nil {
		return nil, fmt.Errorf("unable to parse URL: %v", err)
	}

	if u.Scheme != "s3" {
		return nil, fmt.Errorf("scheme must be s3")
	}

	if params.Region == "" {
		params.Region = "eu-central-1"
	}

	e := EndpointResolver{
		endpointUrl: params.Endpoint,
	}

	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithCredentialsProvider(credsProvider),
		config.WithRegion(params.Region),
		config.WithEndpointResolverWithOptions(e),
	)

	if err != nil {
		return nil, fmt.Errorf("cannot load default config: %v", err)
	}

	s3Client := s3.NewFromConfig(cfg)

	dest.params = &params
	dest.c = s3Client
	dest.bucket = u.Host
	dest.path = u.Path

	if len(dest.path) > 1 {
		dest.path = dest.path[1:]
	} else {
		dest.path = ""
	}

	dest.uploader = manager.NewUploader(s3Client)

	return &dest, nil
}
