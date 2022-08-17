package crawler

import (
	"Search_Engine/pipeline"
	"context"
	"io"
	"net/url"
	"strings"
)

type linkFetcher struct {
	urlGetter   URLGetter
	netDetector PrivateNetworkDetector
}

func newLinkFetcher(urlGetter URLGetter, netDetector PrivateNetworkDetector) *linkFetcher {
	return &linkFetcher{
		urlGetter:   urlGetter,
		netDetector: netDetector,
	}
}

func (lf *linkFetcher) Process(ctx context.Context, p pipeline.Payload) (pipeline.Payload, error) {

	payload := p.(*crawlerPayload)

	// Skips URLs that point to file that cannot contain html content
	if exclusionRegex.MatchString(payload.URL) {
		return nil, nil
	}

	// Never crawl private networks
	if isPrivate, err := lf.isPrivate(payload.URL); err != nil || isPrivate {
		return nil, nil
	}

	res, err := lf.urlGetter.Get(payload.URL)
	if err != nil {
		return nil, nil
	}

	_, err = io.Copy(&payload.RawContent, res.Body)
	_ = res.Body.Close()

	if err != nil {
		return nil, err
	}
	// Skip payloads for invalid http status code
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, nil
	}
	// Skip payloads for non-html payloads
	if contentType := res.Header.Get("Content-Type"); !strings.Contains(contentType, "html") {
		return nil, nil
	}
	return payload, nil
}

func (lf *linkFetcher) isPrivate(URL string) (bool, error) {
	u, err := url.Parse(URL)
	if err != nil {
		return false, err
	}
	return lf.netDetector.isPrivate(u.Hostname())
}
