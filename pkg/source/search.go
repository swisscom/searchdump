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
	"math"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var _ Sourcer = (*Search)(nil)

type Search struct {
	url        *url.URL
	client     *elasticsearch.Client
	logger     *logrus.Logger
	params     SearchParams
	searchInfo *SearchInfo
	verMajor   *int
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

func (s *Search) String() string {
	if s.searchInfo == nil {
		return ""
	}
	return fmt.Sprintf("%s (%s) - %s",
		s.searchInfo.Name,
		s.searchInfo.ClusterName,
		s.searchInfo.Version.Number)
}

func (s *Search) fetchSearchInfo() error {
	res, err := s.client.Info()
	if err != nil {
		return fmt.Errorf("cannot fetch info")
	}

	var esInfo SearchInfo
	dec := json.NewDecoder(res.Body)
	err = dec.Decode(&esInfo)

	if err != nil {
		s.searchInfo = &esInfo
	}

	return err
}

type SearchParams struct {
	IndexFilter string
	Size        int
}

func NewSearch(from string, params *SearchParams, logger *logrus.Logger) (*Search, error) {
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

	if params != nil {
		s.params = *params
	} else {
		s.params = SearchParams{
			IndexFilter: "",
			Size: 50,
		}
	}

	err = s.fetchSearchInfo()
	if err != nil {
		return nil, fmt.Errorf("cannot fetch search info: %v", err)
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

	var re *regexp.Regexp
	if s.params.IndexFilter != "" {
		re, err = regexp.Compile(s.params.IndexFilter)
		if err != nil {
			return nil, fmt.Errorf("unable to compile regexp: %v", err)
		}
	}

	for k, _ := range indicesStats.Indices {
		if re != nil {
			if re.MatchString(k) {
				indices = append(indices, k)
			}
		} else {
			indices = append(indices, k)
		}
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

type Pit struct {
	Id        string `json:"id"`
	KeepAlive string `json:"keep_alive"`
}

type Result struct {
	Took     int    `json:"took"`
	TimedOut bool   `json:"timed_out"`
	Hits     Hits   `json:"hits"`
	Pit      Pit    `json:"pit"`
	ScrollId string `json:"_scroll_id"`
}

type SearchServerError struct {
	StatusCode int
}

func (s SearchServerError) Error() string {
	return fmt.Sprintf("server returned %d", s.StatusCode)
}

var _ error = (*SearchServerError)(nil)

/*
	fetch the documents for the given index and returns:
	- an array of documents for this batch
	- a scroll ID
	- an error (if any)
*/
func (s *Search) fetch(index string, size int, scrollId string) ([]Document, string, error) {
	c := context.Background()
	var res *esapi.Response
	var err error

	scrollTime := 2 * time.Minute

	if scrollId == "" {
		// First is search
		res, err = esapi.SearchRequest{
			Index:  []string{index},
			Size:   &size,
			Scroll: scrollTime,
			Sort:   []string{"_doc"}, // "Scroll requests have optimizations that make them faster when the sort order is _doc."
		}.Do(c, s.client)
	} else {
		res, err = esapi.ScrollRequest{
			ScrollID: scrollId,
			Scroll:   scrollTime,
		}.Do(c, s.client)
	}
	if err != nil {
		return nil, scrollId, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, scrollId, SearchServerError{StatusCode: res.StatusCode}
	}

	var result Result
	j := json.NewDecoder(res.Body)
	err = j.Decode(&result)
	if err != nil {
		return nil, scrollId, fmt.Errorf("unable to decode JSON: %v", err)
	}

	var documents []Document

	for _, e := range result.Hits.HitsEntries {
		documents = append(documents, e.Source)
	}

	return documents, result.ScrollId, nil
}

type PitResponse struct {
	Id string `json:"id"`
}

func (s *Search) versionMajor() int {
	if s.verMajor != nil {
		return *s.verMajor
	}

	if s.searchInfo == nil {
		return -1
	}

	strSplit := strings.Split(s.searchInfo.Version.Number, ".")
	if len(strSplit) == 1 {
		return -1
	}

	major, err := strconv.ParseInt(strSplit[0], 10, 32)
	if err != nil {
		return -1
	}

	majorInt := int(major)
	s.verMajor = &majorInt

	return majorInt
}

/*
	fetchAndSend fetches the documents using the Scroll API
	(ElasticSearch 6.x)

	https://www.elastic.co/guide/en/elasticsearch/reference/6.8/search-request-scroll.html
*/
func (s *Search) fetchAndSend(index string, c chan<- file.File) {
	size := 50
	offset := 0

	s.logger.Infof("fetching index %s", index)

	var indexContent []Document
	currentRetry := 0
	maxRetries := 2

	var scrollId string

	for {
		var entries []Document
		var err error

		entries, scrollId, err = s.fetch(index, size, scrollId)
		if err != nil {
			searchServerError, ok := err.(SearchServerError)
			if currentRetry == maxRetries {
				s.logger.Errorf(
					"search server error: cannot proceed further, our last retry failed: %v. Skipping %s",
					searchServerError,
					index,
				)
				return
			}
			if ok {
				s.logger.Warnf("search server error: %v, retrying offset=%d, size=%d (%d/%d)",
					searchServerError,
					offset,
					size,
					currentRetry,
					maxRetries,
				)
				currentRetry++

				// Sleep for e^(currentRetry) seconds
				d := time.Duration(math.Round(math.Pow(math.E, float64(currentRetry)))) * time.Second
				time.Sleep(d)
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

func (s *Search) do(ch chan<- file.File, indices []string) {
	for _, i := range indices {
		// Fetch data
		s.fetchAndSend(i, ch)
	}

	close(ch)
}

func (s Search) createPit(index string) (interface{}, interface{}) {
	pitRes, err := esapi.OpenPointInTimeRequest{
		Index:     []string{index},
		KeepAlive: "60m",
	}.Do(context.Background(), s.client)
	if err != nil {
		return nil, fmt.Errorf("unable to open PIT: %v", err)
	}

	var pitResponseBody map[string]interface{}
	dec := json.NewDecoder(pitRes.Body)
	err = dec.Decode(&pitResponseBody)
	if err != nil {
		return nil, err
	}

	if pitRes.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cannot create PIT, server returned %d", pitRes.StatusCode)
	}

	return pitResponseBody, nil
}
