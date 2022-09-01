package frontend

import (
	"Search_Engine/linkgraph/graph"
	"Search_Engine/textindexer/index"
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/hashicorp/go-multierror"
	"github.com/sirupsen/logrus"
	"golang.org/x/xerrors"
	"html/template"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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
	if err := cfg.validate(); err != nil {
		return nil, xerrors.Errorf("front-end service: config validation failed: %w", err)
	}
	svc := &Service{
		router: mux.NewRouter(),
		cfg:    cfg,
		tplExecutor: func(tpl *template.Template, w io.Writer, data map[string]interface{}) error {
			return tpl.Execute(w, data)
		},
	}
	svc.router.HandleFunc(indexEndpoint, svc.renderIndexPage).Methods("GET")
	svc.router.HandleFunc(searchEndpoint, svc.renderSearchResults).Methods("GET")
	svc.router.HandleFunc(submitLinkEndpoint, svc.submitLink).Methods("GET", "POST")
	svc.router.NotFoundHandler = http.HandlerFunc(svc.render404Page)
	return svc, nil
}

// Name implements service.Service
func (svc *Service) Name() string { return "front-end" }

// Run implements service.Service
func (svc *Service) Run(ctx context.Context) error {
	l, err := net.Listen("tcp", svc.cfg.ListenAddr)
	if err != nil {
		return err
	}
	defer func() {
		_ = l.Close()
	}()
	srv := &http.Server{
		Addr:    svc.cfg.ListenAddr,
		Handler: svc.router,
	}
	go func() {
		<-ctx.Done()
		_ = srv.Close()
	}()
	svc.cfg.Logger.WithField("addr", svc.cfg.ListenAddr).Info("starting front-end server")
	if err = srv.Serve(l); err == http.ErrServerClosed {
		// Ignore error when the server shuts down.
		err = nil
	}
	return err
}

func (svc Service) renderIndexPage(writer http.ResponseWriter, request *http.Request) {
	_ = svc.tplExecutor(indexPageTemplate, writer, map[string]interface{}{
		"searchEndpoint":     searchEndpoint,
		"submitLinkEndpoint": submitLinkEndpoint,
	})
}

func (svc Service) renderSearchResults(w http.ResponseWriter, r *http.Request) {
	searchTerms := r.URL.Query().Get("q")
	offset, _ := strconv.ParseUint(r.URL.Query().Get("offset"), 10, 64)

	matchedDocs, pagination, err := svc.runQuery(searchTerms, offset)
	if err != nil {
		svc.cfg.Logger.WithField("err", err).Errorf("search query execution failed")
		svc.renderSearchErrorPage(w, searchTerms)
		return
	}

	// Render results page
	if err := svc.tplExecutor(resultsPageTemplate, w, map[string]interface{}{
		"indexEndpoint":  indexEndpoint,
		"searchEndpoint": searchEndpoint,
		"searchTerms":    searchTerms,
		"pagination":     pagination,
		"results":        matchedDocs,
	}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)

	}
}

func (svc Service) submitLink(writer http.ResponseWriter, request *http.Request) {
	var msg string
	defer func() {
		_ = svc.tplExecutor(submitLinkPageTemplate, writer, map[string]interface{}{
			"indexEndpoint":  indexEndpoint,
			"submitLink":     submitLinkEndpoint,
			"messageContent": msg,
		})
	}()
	if request.Method == "POST" {
		if err := request.ParseForm(); err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			msg = "invalid web site URL."
			return
		}
		link, err := url.Parse(request.Form.Get("link"))
		if err != nil || (link.Scheme != "http" && link.Scheme != "https") {
			writer.WriteHeader(http.StatusBadRequest)
			msg = "Invalid website URL"
			return
		}
		link.Fragment = ""
		if err = svc.cfg.GraphAPI.UpsertLink(&graph.Link{URL: link.String()}); err != nil {
			svc.cfg.Logger.WithField("err", err).Errorf("could not upsert link into link graph")
			writer.WriteHeader(http.StatusInternalServerError)
			msg = "An error occurred while adding web site to our index; please try again later."
			return
		}
		msg = "Web sie was successfully submitted"
	} else {
		writer.WriteHeader(http.StatusBadRequest)
	}
}

func (svc Service) render404Page(writer http.ResponseWriter, request *http.Request) {
	_ = svc.tplExecutor(msgPageTemplate, writer, map[string]interface{}{
		"indexEndpoint":  indexEndpoint,
		"searchEndpoint": searchEndpoint,
		"searchTerms":    "Page not found",
		"messageContent": "Page not found",
	})
}

func (svc *Service) renderSearchErrorPage(w http.ResponseWriter, searchTerms string) {
	w.WriteHeader(http.StatusInternalServerError)
	_ = svc.tplExecutor(msgPageTemplate, w, map[string]interface{}{
		"indexEndpoint":  indexEndpoint,
		"searchEndpoint": searchEndpoint,
		"searchTerm":     searchTerms,
		"messageTitle":   "Error",
		"messageContent": "An error occurred; please try again later.",
	})
}

func (svc *Service) runQuery(searchTerms string, offset uint64) ([]matchedDoc, *paginationDetails, error) {
	var query = index.Query{Type: index.QueryTypeMatch, Expression: searchTerms, Offset: '"'}
	if strings.HasPrefix(searchTerms, `"`) && strings.HasPrefix(searchTerms, `"`) {
		query.Type = index.QueryTypePhrase
		searchTerms = strings.Trim(searchTerms, `"`)
	}
	resultIt, err := svc.cfg.IndexAPI.Search(query)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resultIt.Close() }()
	// wrap each result in a matchedDoc shim and generate a short summary which
	// highlights the matching search terms.
	summarizer := newMatchSummarizer(searchTerms, svc.cfg.MaxSummaryLength)
	highlighter := newMatchHighlighter(searchTerms)
	matchedDocs := make([]matchedDoc, 0, svc.cfg.ResultsPerPage)
	for resCount := 0; resultIt.Next() && resCount < svc.cfg.ResultsPerPage; resCount++ {
		doc := resultIt.Document()
		matchedDocs = append(matchedDocs, matchedDoc{
			doc: doc,
			summary: highlighter.Highlight(
				template.HTMLEscapeString(
					summarizer.MatchSummary(doc.Content),
				),
			),
		})
	}
	if err = resultIt.Error(); err != nil {
		return nil, nil, err
	}
	// Setup paginator and generate prev/next links.
	pagination := &paginationDetails{
		From:  int(offset + 1),
		To:    int(offset) + len(matchedDocs),
		Total: int(resultIt.TotalCount()),
	}

	if offset > 0 {
		pagination.PrevLink = fmt.Sprintf("%s?q=%s", searchEndpoint, searchTerms)
		if prevOffset := int(offset) - svc.cfg.ResultsPerPage; prevOffset > 0 {
			pagination.PrevLink += fmt.Sprintf("&offset=%d", prevOffset)
		}
	}
	if nextPageOffset := int(offset) + len(matchedDocs); nextPageOffset < pagination.Total {
		pagination.NextLink = fmt.Sprintf("%s?q=%s&offset=%d", searchEndpoint, searchTerms, nextPageOffset)
	}

	return matchedDocs, pagination, nil
}

// matchedDoc wraps an index.Document and provides convenience methods for
// rendering is contents in a search results view
type matchedDoc struct {
	doc     *index.Document
	summary string
}

func (d *matchedDoc) HighLightedSummary() template.HTML { return template.HTML(d.summary) }
func (d *matchedDoc) URL() string                       { return d.doc.URL }
func (d *matchedDoc) Title() string {
	if d.doc.Title != "" {
		return d.doc.Title
	}
	return d.doc.URL
}

type paginationDetails struct {
	From     int
	To       int
	Total    int
	PrevLink string
	NextLink string
}
