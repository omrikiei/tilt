package assets

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/windmilleng/tilt/pkg/model"
)

const (
	testUrl        = "https://fake.tilt.dev"
	versionDefault = model.WebVersion("v1.2.3")
	version666     = model.WebVersion("v6.6.6")
)

func TestBuildUrlForReq(t *testing.T) {
	s := prodServerForTest(t)
	expected := "https://fake.tilt.dev/v1.2.3/index.html"
	req := reqForTest(t, "/", "")
	u, v := s.urlAndVersionForReq(req)
	assert.Equal(t, expected, u.String())
	assert.Equal(t, string(versionDefault), v)
}

func TestBuildUrlForReqRedirectsToIndex(t *testing.T) {
	s := prodServerForTest(t)
	expected := "https://fake.tilt.dev/v1.2.3/index.html"
	req := reqForTest(t, "/some/random/path", "")
	u, v := s.urlAndVersionForReq(req)
	assert.Equal(t, expected, u.String())
	assert.Equal(t, string(versionDefault), v)
}

func TestBuildUrlForReqRespectsStatic(t *testing.T) {
	s := prodServerForTest(t)
	expected := "https://fake.tilt.dev/v1.2.3/static/stuff.html"
	req := reqForTest(t, "/static/stuff.html", "")
	u, v := s.urlAndVersionForReq(req)
	assert.Equal(t, expected, u.String())
	assert.Equal(t, string(versionDefault), v)
}

func TestBuildUrlForReqRespectsVersion(t *testing.T) {
	s := prodServerForTest(t)
	expected := "https://fake.tilt.dev/v111.222.333/stuff.html"
	req := reqForTest(t, "/v111.222.333/stuff.html", "")
	u, v := s.urlAndVersionForReq(req)
	assert.Equal(t, expected, u.String())
	assert.Equal(t, "v111.222.333", v)
}

func TestBuildUrlForReqWithVersionParam(t *testing.T) {
	s := prodServerForTest(t)
	expected := "https://fake.tilt.dev/v6.6.6/index.html"
	req := reqForTest(t, "/", version666)
	u, v := s.urlAndVersionForReq(req)
	assert.Equal(t, expected, u.String())
	assert.Equal(t, string(version666), v)
}

func TestBuildUrlForReqWithVersionParamAndStaticPath(t *testing.T) {
	s := prodServerForTest(t)
	expected := "https://fake.tilt.dev/v6.6.6/static/stuff.html"
	req := reqForTest(t, "/static/stuff.html", version666)
	u, v := s.urlAndVersionForReq(req)
	assert.Equal(t, expected, u.String())
	assert.Equal(t, string(version666), v)
}

func TestBuildUrlForReqWithVersionParamAndVersionPrefix(t *testing.T) {
	s := prodServerForTest(t)
	expected := "https://fake.tilt.dev/v111.222.333/stuff.html"
	req := reqForTest(t, "/v111.222.333/stuff.html", version666)
	u, v := s.urlAndVersionForReq(req)
	assert.Equal(t, expected, u.String())
	assert.Equal(t, "v111.222.333", v)
}

func prodServerForTest(t *testing.T) prodServer {
	s, err := newProdServer(testUrl, versionDefault)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func reqForTest(t *testing.T, path string, version model.WebVersion) *http.Request {
	u, err := url.Parse(path)
	if err != nil {
		t.Fatal(err)
	}

	if version != "" {
		q := u.Query()
		q.Set(WebVersionKey, string(version))
		u.RawQuery = q.Encode()
	}

	return &http.Request{URL: u}
}
