module github.com/swisscom/searchdump

go 1.17

require (
	github.com/alexflint/go-arg v1.4.2
	github.com/aws/aws-sdk-go-v2/config v1.11.1
	github.com/aws/aws-sdk-go-v2/credentials v1.6.5
	github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.7.5
	github.com/aws/aws-sdk-go-v2/service/s3 v1.22.0
	github.com/elastic/go-elasticsearch/v8 v8.0.0-20210817150010-57d659deaca7
	github.com/sirupsen/logrus v1.8.1
)

require (
	github.com/alexflint/go-scalar v1.0.0 // indirect
	github.com/aws/aws-sdk-go-v2 v1.11.2 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.0.0 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.8.2 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.2 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.0.2 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.5.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.5.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.9.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.7.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.12.0 // indirect
	github.com/aws/smithy-go v1.9.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	golang.org/x/sys v0.0.0-20210423082822-04245dca01da // indirect
)

replace github.com/elastic/go-elasticsearch/v8 => github.com/denysvitali/go-elasticsearch/v8 v8.0.0-20210913230432-3aa1a31cc790
