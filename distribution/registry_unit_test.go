package distribution // import "github.com/docker/docker/distribution"

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/containerd/log"
	"github.com/distribution/reference"
	"github.com/docker/docker/api/types/registry"
	registrypkg "github.com/docker/docker/registry"
)

const secretRegistryToken = "mysecrettoken"

type tokenPassThruHandler struct {
	reached       bool
	gotToken      bool
	shouldSend401 func(url string) bool
}

func (h *tokenPassThruHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.reached = true
	if strings.Contains(r.Header.Get("Authorization"), secretRegistryToken) {
		log.G(context.TODO()).Debug("Detected registry token in auth header")
		h.gotToken = true
	}
	if h.shouldSend401 == nil || h.shouldSend401(r.RequestURI) {
		w.Header().Set("WWW-Authenticate", `Bearer realm="foorealm"`)
		w.WriteHeader(401)
	}
}

func testTokenPassThru(t *testing.T, ts *httptest.Server) {
	uri, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatalf("could not parse url from test server: %v", err)
	}

	repoName, _ := reference.ParseNormalizedNamed("testremotename")
	imagePullConfig := &ImagePullConfig{
		Config: Config{
			MetaHeaders: http.Header{},
			AuthConfig: &registry.AuthConfig{
				RegistryToken: secretRegistryToken,
			},
		},
	}
	p := newPuller(registrypkg.APIEndpoint{URL: uri}, repoName, imagePullConfig, nil)
	ctx := context.Background()
	p.repo, err = newRepository(ctx, repoName, p.endpoint, p.config.MetaHeaders, p.config.AuthConfig, "pull")
	if err != nil {
		t.Fatal(err)
	}

	log.G(ctx).Debug("About to pull")
	// We expect it to fail, since we haven't mock'd the full registry exchange in our handler above
	tag, _ := reference.WithTag(repoName, "tag_goes_here")
	_ = p.pullRepository(ctx, tag)
}

func TestTokenPassThru(t *testing.T) {
	handler := &tokenPassThruHandler{shouldSend401: func(url string) bool { return url == "/v2/" }}
	ts := httptest.NewServer(handler)
	defer ts.Close()

	testTokenPassThru(t, ts)

	if !handler.reached {
		t.Fatal("Handler not reached")
	}
	if !handler.gotToken {
		t.Fatal("Failed to receive registry token")
	}
}

func TestTokenPassThruDifferentHost(t *testing.T) {
	handler := new(tokenPassThruHandler)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	tsredirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/v2/" {
			w.Header().Set("WWW-Authenticate", `Bearer realm="foorealm"`)
			w.WriteHeader(401)
			return
		}
		http.Redirect(w, r, ts.URL+r.URL.Path, http.StatusMovedPermanently)
	}))
	defer tsredirect.Close()

	testTokenPassThru(t, tsredirect)

	if !handler.reached {
		t.Fatal("Handler not reached")
	}
	if handler.gotToken {
		t.Fatal("Redirect should not forward Authorization header to another host")
	}
}
