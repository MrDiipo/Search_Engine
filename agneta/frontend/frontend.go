package frontend

import (
	"Search_Engine/linkgraph/graph"
	"Search_Engine/textindexer/index"
	"github.com/gorilla/mux"
	"github.com/hashicorp/go-multierror"
	"github.com/sirupsen/logrus"
	"golang.org/x/xerrors"
	"html/template"
	"io"
	"io/ioutil"
)

const (
	indexEndpoint      = "/"
	searchEndpoint     = "/search"
	submitLinkEndpoint = "/submit/site"

	defaultResultsPerPage   = 10
	defaultMaxSummaryLength = 256
)

// GraphAPI defines a set of API methods for adding links to the graph.
type GraphAPI interface {
	UpsertLink(link *graph.Link) error
}

// IndexAPI defines a set of API methods for searching crawled documents.
type IndexAPI interface {
	Search(query index.Query) (index.Iterator, error)
}

// Config encapsulates the settings for configuring the front-end service.
type Config struct {
	// An API for adding links to the link graph.
	GraphAPI GraphAPI

	// An API for executing queries against indexed documents.
	IndexAPI IndexAPI

	// The port to listen for incoming requests.
	ListenAddr string

	// The number of results to display per page. If not specified, a default
	// value of 10 results per page will be used instead.
	ResultsPerPage int

	// The maximum length (in characters) of the highlighted content summary for
	// matching documents. If not specified, a default value of 256 will be used
	// instead.
	MaxSummaryLength int

	// The logger to use. If not defined an output-discarding logger will
	// be used instead.
	Logger *logrus.Entry
}

func (cfg *Config) validate() error {
	var err error
	if cfg.ListenAddr == "" {
		err = multierror.Append(err, xerrors.Errorf("listen address has not been specified"))
	}
	if cfg.ResultsPerPage <= 0 {
		cfg.ResultsPerPage = defaultResultsPerPage
	}
	if cfg.MaxSummaryLength <= 0 {
		cfg.MaxSummaryLength = defaultMaxSummaryLength
	}
	if cfg.IndexAPI == nil {
		err = multierror.Append(err, xerrors.Errorf("index API has not been provided"))
	}
	if cfg.GraphAPI == nil {
		err = multierror.Append(err, xerrors.Errorf("graph API has not been provided"))
	}
	if cfg.Logger == nil {
		cfg.Logger = logrus.NewEntry(&logrus.Logger{Out: ioutil.Discard})
	}
	return err
}

// Service implements the front-end component for the Agneta Search Engine.
type Service struct {
	cfg    Config
	router *mux.Router
	// A template executor hook
	tplExecutor func(tpl *template.Template, w io.Writer, data map[string]interface{}) error
}

func NewService(cfg Config) (*Service, error) {
	if err := cfg.validate();
}