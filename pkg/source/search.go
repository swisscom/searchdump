package source

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/sirupsen/logrus"
	"github.com/swisscom/searchdump/pkg/file"
	"net/http"
	"net/url"
	"sort"
	"time"
)

var _ Sourcer = (*Search)(nil)

type Search struct {
	url    *url.URL
	client *elasticsearch.Client
	logger *logrus.Logger
}

type Version struct {
	Number                           string    `json:"number"`
	BuildFlavor                      string    `json:"build_flavor"`
	BuildType                        string    `json:"build_type"`
	BuildHash                        string    `json:"build_hash"`
	BuildDate                        time.Time `json:"build_date"`
	BuildSnapshot                    bool      `json:"build_snapshot"`
	LuceneVersion                    string    `json:"lucene_version"`
	MinimumWireCompatibilityVersion  string    `json:"minimum_wire_compatibility_version"`
	MinimumIndexCompatibilityVersion string    `json:"minimum_index_compatibility_version"`
}

type SearchInfo struct {
	Name        string  `json:"name"`
	ClusterName string  `json:"cluster_name"`
	ClusterUuid string  `json:"cluster_uuid"`
	Version     Version `json:"version"`
	Tagline     string  `json:"tagline"`
}

func (s Search) String() string {
	if s.client == nil {
		return ""
	}

	res, err := s.client.Info()
	if err != nil {
		return fmt.Sprintf("cannot fetch info")
	}

	var esInfo SearchInfo
	dec := json.NewDecoder(res.Body)
	err = dec.Decode(&esInfo)

	if err != nil {
		return "Search (no info)"
	}

	return fmt.Sprintf("%s (%s) - %s", esInfo.Name, esInfo.ClusterName, esInfo.Version.Number)
}

func NewSearch(from string, logger *logrus.Logger) (*Search, error) {
	u, err := url.Parse(from)
	if err != nil {
		return nil, fmt.Errorf("unable to parse URL: %v", err)
	}

	password, _ := u.User.Password()

	cfg := elasticsearch.Config{
		Addresses: []string{
			fmt.Sprintf("%s://%s", u.Scheme, u.Host),
		},
		Username: u.User.Username(),
		Password: password,
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("cannot create ElasticSearch client: %v", err)
	}

	if logger == nil {
		logger = logrus.New()
	}

	s := Search{
		url:    u,
		client: client,
		logger: logger,
	}

	return &s, nil
}

type Shards struct {
	Total      int `json:"total"`
	Successful int `json:"successful"`
	Failed     int `json:"failed"`
}

type IndicesStatsAll struct {
	Primaries map[string]interface{} `json:"primaries"`
	Total     map[string]interface{} `json:"total"`
}

type IndicesStats struct {
	Shards  Shards                 `json:"_shards"`
	All     IndicesStatsAll        `json:"_all"`
	Indices map[string]interface{} `json:"indices"`
}

func (s *Search) getIndices() ([]string, error) {
	var indices []string
	ctx := context.Background()
	res, err := esapi.IndicesStatsRequest{}.Do(ctx, s.client)
	if err != nil {
		return indices, err
	}
	defer res.Body.Close()
	var indicesStats IndicesStats
	dec := json.NewDecoder(res.Body)
	err = dec.Decode(&indicesStats)
	if err != nil {
		return nil, err
	}

	for k, _ := range indicesStats.Indices {
		indices = append(indices, k)
	}

	sort.Strings(indices)
	return indices, nil
}

func (s *Search) Fetch() (chan file.File, error) {
	c := make(chan file.File)
	// Get indices
	indices, err := s.getIndices()
	if err != nil {
		return c, fmt.Errorf("unable to fetch indices: %v", err)
	}

	go s.do(c, indices)
	return c, nil
}

type HitsTotal struct {
	Value    int    `json:"value"`
	Relation string `json:"relation"`
}

type Document map[string]interface{}

type HitsEntry struct {
	Index  string   `json:"_index"`
	Type   string   `json:"_type"`
	Id     string   `json:"_id"`
	Score  float64  `json:"_score"`
	Source Document `json:"_source"`
}

type Hits struct {
	Total       int         `json:"total"`
	HitsEntries []HitsEntry `json:"hits"`
}

type Result struct {
	Took     int  `json:"took"`
	TimedOut bool `json:"timed_out"`
	Hits     Hits `json:"hits"`
}

type SearchServerError struct {
	StatusCode int
}

func (s SearchServerError) Error() string {
	return fmt.Sprintf("server returned %d", s.StatusCode)
}

var _ error = (*SearchServerError)(nil)

func (s *Search) fetch(index string, size int, offset int) ([]Document, error) {
	c := context.Background()
	res, err := esapi.SearchRequest{
		Index: []string{index},
		From:  &offset,
		Size:  &size,
	}.Do(c, s.client)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, SearchServerError{StatusCode: res.StatusCode}
	}

	var result Result
	j := json.NewDecoder(res.Body)
	err = j.Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("unable to decode JSON: %v", err)
	}

	var documents []Document

	for _, e := range result.Hits.HitsEntries {
		documents = append(documents, e.Source)
	}

	return documents, nil
}

func (s *Search) fetchAndSend(index string, c chan <- file.File) {
	size := 50
	offset := 0

	s.logger.Infof("fetching index %s", index)

	var indexContent []Document
	currentRetry := 0
	maxRetries := 4

	for {
		entries, err := s.fetch(index, size, offset)
		if err != nil {
			searchServerError, ok := err.(SearchServerError)
			if currentRetry == maxRetries {
				s.logger.Fatalf("search server error: cannot proceed further, our last retry failed: %v", searchServerError)
			}
			if ok {
				s.logger.Warnf("search server error: %v, retrying (%d/%d)",
					searchServerError,
					currentRetry,
					maxRetries,
				)
				currentRetry++
				continue
			}
			s.logger.Fatalf("unable to fetch: index=%s, size=%d, offset=%d: %v", index, size, offset, err)
		}

		currentRetry = 0

		if len(entries) == 0 {
			break
		}
		indexContent = append(indexContent, entries...)
		offset += size
	}

	marshal, err := json.Marshal(&indexContent)
	if err != nil {
		s.logger.Fatalf("unable to marshal JSON: %v", err)
	}

	c <- file.File{
		Name:    fmt.Sprintf("%s.json", index),
		Content: bytes.NewReader(marshal),
	}
}

func (s *Search) do(ch chan <- file.File, indices []string) {
	for _, i := range indices {
		// Fetch data
		s.fetchAndSend(i, ch)
	}

	close(ch)
}
