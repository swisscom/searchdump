package searchdump

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/swisscom/searchdump/pkg/dest"
	"github.com/swisscom/searchdump/pkg/source"
	"strings"
)

type Client struct {
	From     source.Sourcer
	FromType FromType

	To     dest.Dester
	ToType ToType
	logger *logrus.Logger
}

func New() Client {
	return Client{
		FromType: FromNone,
		To: dest.NoneDest{},
		logger: logrus.New(),
	}
}

func (c *Client) SetFrom(fromType string, from string) error {
	switch strings.ToLower(fromType) {
	case "elasticsearch", "opensearch", "search":
		var err error
		c.FromType = FromSearch
		c.From, err = source.NewSearch(from, c.logger)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown from type \"%s\"", fromType)
	}

	return nil
}

func (c *Client) SetLogger(logger *logrus.Logger) {
	if logger == nil {
		return
	}
	c.logger = logger
}

func (c *Client) Start() error {
	c.logger.Infof("Source: %s", c.From)
	c.logger.Infof("Dest: %s", c.To)

	ch, err := c.From.Fetch()

	for f := range ch {
		err = c.To.Write(f)
		if err != nil {
			c.logger.Fatalf("unable to write file %s", f.Name)
			break
		}
		c.logger.Infof("wrote %s", f.Name)
	}


	if err != nil {
		return fmt.Errorf("unable to start fetching: %v", err)
	}
	return nil
}

func (c *Client) SetTo(toType string, params interface{}) error {
	switch strings.ToLower(toType) {
	case "s3":
		v, ok := params.(dest.S3Params)
		if !ok {
			return fmt.Errorf("cannot cast to S3Params")
		}


		s3Dest, err := dest.NewS3(v)
		if err != nil {
			return err
		}
		c.To = s3Dest
	}

	return nil
}
