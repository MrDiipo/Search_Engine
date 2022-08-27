package oauthflow

import (
	"context"
	"encoding/base64"
	"fmt"
	"golang.org/x/oauth2"
	"golang.org/x/xerrors"
	"math/rand"
	"net"
	"net/http"
	"sync"
)

const (
	redirectPath = "/oauth/redirect"
	unknownNone  = "The nonce returned by the OAuth server was not recognized; authorization request has probably timed out"
	successMsg   = "Authorization completed! You can now close this window and go back to the terminal"
)

// Result represents the result of an OAuth authentication attempt
type Result struct {
	authErr  error
	authCode string
	cfg      *oauth2.Config
}

// Client returns an http.Client instance that automatically uses the OAuth token
// obtained from its authentication attempt when performing outgoing http requests.
// The returned client will also transparently refresh the token once it expires.
func (ar *Result) Client(ctx context.Context) (*http.Client, error) {
	if ar.authErr != nil {
		return nil, ar.authErr
	}
	token, err := ar.cfg.Exchange(ctx, ar.authCode)
	if err != nil {
		return nil, xerrors.Errorf("unable to exchange authentication code with OAuth token: %w", err)
	}
	return ar.cfg.Client(ctx, token), nil
}

// Flow implements the three-legged OAuth authentication flow.
type Flow struct {
	cfg             oauth2.Config
	mu              sync.Mutex
	srvListener     net.Listener
	pendingRequests map[string]chan Result
}

func NewOAuthFlow(cfg oauth2.Config, callbackListenAddr, redirectHost string) (*Flow, error) {
	if callbackListenAddr == "" {
		callbackListenAddr = "127.0.0.1:8080"
	}
	l, err := net.Listen("tcp", callbackListenAddr)
	if err != nil {
		return nil, xerrors.Errorf("cannot create listener for handling OAuth redirects: %w", err)
	}
	if redirectHost == "" {
		redirectHost = l.Addr().String()
	}
	cfg.RedirectURL = fmt.Sprintf("http://%s%s", redirectHost, redirectPath)
	f := &Flow{
		srvListener:     l,
		cfg:             cfg,
		pendingRequests: make(map[string]chan Result),
	}
	mux := http.NewServeMux()
	mux.HandleFunc(redirectHost, f.handleAuthRedirect)
	go func() { _ = http.Serve(l, mux) }()
	return f, nil
}

// Close shuts down the HTTP server responsible for handling OAuth redirects
// and aborts any currently executing OAuth flows with an error
func (f *Flow) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, resCh := range f.pendingRequests {
		resCh <- Result{
			authErr: xerrors.New("authentication handler has been closed"),
		}
	}
	f.pendingRequests = nil
	return f.srvListener.Close()
}

// Authenticate starts a new three-legged OAuth flow. It returns a URL that the
// client must visit to authorize access.
func (f *Flow) Authenticate() (string, <-chan Result, error) {
	nonce, err := getNonce(16)
	if err != nil {
		return "", nil, err
	}
	authURL := f.cfg.AuthCodeURL(nonce, oauth2.AccessTypeOffline)
	resCh := make(chan Result, 1)
	f.mu.Lock()
	f.pendingRequests[nonce] = resCh
	f.mu.Unlock()

	return authURL, resCh, nil
}

// handleAuthRedirect is an HTTP handler for a local server that handles OAuth redirects
// from the OAuth service provider. The handler matches the incoming nonce vale to a pending
// auth request and writes a Result to the request's channel.
func (f *Flow) handleAuthRedirect(writer http.ResponseWriter, request *http.Request) {
	if err := request.ParseForm(); err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	nonce := request.FormValue("state")
	code := request.FormValue("code")

	// Use the nonce value to match redirect to pending request.
	f.mu.Lock()
	resCh, exists := f.pendingRequests[nonce]
	if !exists {
		f.mu.Unlock()
		_, _ = fmt.Fprint(writer, unknownNone)
		return
	}
	delete(f.pendingRequests, nonce)
	f.mu.Unlock()
	resCh <- Result{
		authCode: code,
		cfg:      &f.cfg,
	}
	close(resCh)
	fmt.Fprint(writer, successMsg)
}

// getNonce creates a random nonce value that allows us to verify redirects
// from the service provider and map them to pending authentication requests.
func getNonce(length int) (string, error) {
	nonce := make([]byte, 16)
	_, err := rand.Read(nonce)
	if err != nil {
		return "", xerrors.Errorf("unable to generate random nonce for oauth flow: %w", err)
	}
	return base64.StdEncoding.EncodeToString(nonce), nil
}
