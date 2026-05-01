package http

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mpolden/echoip/iputil/geo"
)

func lookupAddr(net.IP) (string, error) { return "localhost", nil }
func lookupPort(net.IP, uint64) error   { return nil }

type testDb struct{}
type ipTestCase struct {
	remoteAddr     string
	headerKey      string
	headerValue    string
	trustedHeaders []string
	out            string
}

func (t *testDb) Country(net.IP) (geo.Country, error) {
	return geo.Country{Name: "Elbonia", ISO: "EB"}, nil
}

func (t *testDb) City(net.IP) (geo.City, error) {
	return geo.City{Name: "Bornyasherk", RegionName: "North Elbonia", RegionCode: "1234", MetroCode: 1234, PostalCode: "1234", Latitude: 63.416667, Longitude: 10.416667, Timezone: "Europe/Bornyasherk"}, nil
}

func (t *testDb) ASN(net.IP) (geo.ASN, error) {
	return geo.ASN{AutonomousSystemNumber: 59795, AutonomousSystemOrganization: "Hosting4Real"}, nil
}

func (t *testDb) IsEmpty() bool { return false }

func testServer() *Server {
	return &Server{cache: NewCache(100), gr: &testDb{}, LookupAddr: lookupAddr, LookupPort: lookupPort}
}

func httpGet(url string, acceptMediaType string, userAgent string) (string, int, error) {
	r, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", 0, err
	}
	if acceptMediaType != "" {
		r.Header.Set("Accept", acceptMediaType)
	}
	r.Header.Set("User-Agent", userAgent)
	res, err := http.DefaultClient.Do(r)
	if err != nil {
		return "", 0, err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return "", 0, err
	}
	return string(data), res.StatusCode, nil
}

func httpPost(url, body string) (*http.Response, string, error) {
	r, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return nil, "", err
	}
	res, err := http.DefaultClient.Do(r)
	if err != nil {
		return nil, "", err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, "", err
	}
	return res, string(data), nil
}

func TestCLIHandlers(t *testing.T) {
	log.SetOutput(io.Discard)
	s := httptest.NewServer(testServer().Handler())

	var tests = []struct {
		url             string
		out             string
		status          int
		userAgent       string
		acceptMediaType string
	}{
		{s.URL, "127.0.0.1\n", 200, "curl/7.43.0", ""},
		{s.URL, "127.0.0.1\n", 200, "foo/bar", textMediaType},
		{s.URL + "/ip", "127.0.0.1\n", 200, "", ""},
		{s.URL + "/country", "Elbonia\n", 200, "", ""},
		{s.URL + "/country-iso", "EB\n", 200, "", ""},
		{s.URL + "/coordinates", "63.416667,10.416667\n", 200, "", ""},
		{s.URL + "/city", "Bornyasherk\n", 200, "", ""},
		{s.URL + "/foo", "404 page not found", 404, "", ""},
		{s.URL + "/asn", "AS59795\n", 200, "", ""},
		{s.URL + "/asn-org", "Hosting4Real\n", 200, "", ""},
	}

	for _, tt := range tests {
		out, status, err := httpGet(tt.url, tt.acceptMediaType, tt.userAgent)
		if err != nil {
			t.Fatal(err)
		}
		if status != tt.status {
			t.Errorf("Expected %d, got %d", tt.status, status)
		}
		if out != tt.out {
			t.Errorf("Expected %q, got %q", tt.out, out)
		}
	}
}

func TestDisabledHandlers(t *testing.T) {
	log.SetOutput(io.Discard)
	server := testServer()
	server.LookupPort = nil
	server.LookupAddr = nil
	server.gr, _ = geo.Open("", "", "")
	s := httptest.NewServer(server.Handler())

	var tests = []struct {
		url    string
		out    string
		status int
	}{
		{s.URL + "/port/1337", "404 page not found", 404},
		{s.URL + "/country", "404 page not found", 404},
		{s.URL + "/country-iso", "404 page not found", 404},
		{s.URL + "/city", "404 page not found", 404},
		{s.URL + "/json", "{\n  \"ip\": \"127.0.0.1\",\n  \"ip_decimal\": 2130706433,\n  \"country_eu\": false,\n  \"is_proxy\": false,\n  \"headers\": {\n    \"Accept-Encoding\": \"gzip\"\n  }\n}", 200},
	}

	for _, tt := range tests {
		out, status, err := httpGet(tt.url, "", "")
		if err != nil {
			t.Fatal(err)
		}
		if status != tt.status {
			t.Errorf("Expected %d, got %d", tt.status, status)
		}
		if out != tt.out {
			t.Errorf("Expected %q, got %q", tt.out, out)
		}
	}
}

func TestJSONHandlers(t *testing.T) {
	log.SetOutput(io.Discard)
	s := httptest.NewServer(testServer().Handler())

	var tests = []struct {
		url    string
		out    string
		status int
	}{
		{s.URL, "{\n  \"ip\": \"127.0.0.1\",\n  \"ip_decimal\": 2130706433,\n  \"country\": \"Elbonia\",\n  \"country_iso\": \"EB\",\n  \"country_eu\": false,\n  \"region_name\": \"North Elbonia\",\n  \"region_code\": \"1234\",\n  \"metro_code\": 1234,\n  \"zip_code\": \"1234\",\n  \"city\": \"Bornyasherk\",\n  \"latitude\": 63.416667,\n  \"longitude\": 10.416667,\n  \"time_zone\": \"Europe/Bornyasherk\",\n  \"asn\": \"AS59795\",\n  \"asn_org\": \"Hosting4Real\",\n  \"hostname\": \"localhost\",\n  \"user_agent\": {\n    \"product\": \"curl\",\n    \"version\": \"7.2.6.0\",\n    \"raw_value\": \"curl/7.2.6.0\"\n  },\n  \"is_proxy\": false,\n  \"headers\": {\n    \"Accept\": \"application/json\",\n    \"Accept-Encoding\": \"gzip\",\n    \"User-Agent\": \"curl/7.2.6.0\"\n  }\n}", 200},
		{s.URL + "/port/foo", "{\n  \"status\": 400,\n  \"error\": \"invalid port: foo\"\n}", 400},
		{s.URL + "/port/0", "{\n  \"status\": 400,\n  \"error\": \"invalid port: 0\"\n}", 400},
		{s.URL + "/port/65537", "{\n  \"status\": 400,\n  \"error\": \"invalid port: 65537\"\n}", 400},
		{s.URL + "/port/31337", "{\n  \"ip\": \"127.0.0.1\",\n  \"port\": 31337,\n  \"reachable\": true\n}", 200},
		{s.URL + "/port/80", "{\n  \"ip\": \"127.0.0.1\",\n  \"port\": 80,\n  \"reachable\": true\n}", 200},            // checking that our test server is reachable on port 80
		{s.URL + "/port/80?ip=1.3.3.7", "{\n  \"ip\": \"127.0.0.1\",\n  \"port\": 80,\n  \"reachable\": true\n}", 200}, // ensuring that the "ip" parameter is not usable to check remote host ports
		{s.URL + "/foo", "{\n  \"status\": 404,\n  \"error\": \"404 page not found\"\n}", 404},
		{s.URL + "/health", `{"status":"OK"}`, 200},
	}

	for _, tt := range tests {
		out, status, err := httpGet(tt.url, jsonMediaType, "curl/7.2.6.0")
		if err != nil {
			t.Fatal(err)
		}
		if status != tt.status {
			t.Errorf("Expected %d for %s, got %d", tt.status, tt.url, status)
		}
		if out != tt.out {
			t.Errorf("Expected %q for %s, got %q", tt.out, tt.url, out)
		}
	}
}

func TestCacheHandler(t *testing.T) {
	log.SetOutput(io.Discard)
	srv := testServer()
	srv.profile = true
	s := httptest.NewServer(srv.Handler())
	got, _, err := httpGet(s.URL+"/debug/cache/", jsonMediaType, "")
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"size\": 0,\n  \"capacity\": 100,\n  \"evictions\": 0\n}"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCacheResizeHandler(t *testing.T) {
	log.SetOutput(io.Discard)
	srv := testServer()
	srv.profile = true
	s := httptest.NewServer(srv.Handler())
	_, got, err := httpPost(s.URL+"/debug/cache/resize", "10")
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"message\": \"Changed cache capacity to 10.\"\n}"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestIPv4FromRequest(t *testing.T) {
	var tests = []ipTestCase{
		{"127.0.0.1:9999", "", "", nil, "127.0.0.1"},                                                                // No header given
		{"127.0.0.1:9999", "X-Real-IP", "1.3.3.7", nil, "127.0.0.1"},                                                // Trusted header is empty
		{"127.0.0.1:9999", "X-Real-IP", "1.3.3.7", []string{"X-Foo-Bar"}, "127.0.0.1"},                              // Trusted header does not match
		{"127.0.0.1:9999", "X-Real-IP", "1.3.3.7", []string{"X-Real-IP", "X-Forwarded-For"}, "1.3.3.7"},             // Trusted header matches
		{"127.0.0.1:9999", "X-Forwarded-For", "1.3.3.7", []string{"X-Real-IP", "X-Forwarded-For"}, "1.3.3.7"},       // Second trusted header matches
		{"127.0.0.1:9999", "X-Forwarded-For", "1.3.3.7,4.2.4.2", []string{"X-Forwarded-For"}, "1.3.3.7"},            // X-Forwarded-For with multiple entries (commas separator)
		{"127.0.0.1:9999", "X-Forwarded-For", "1.3.3.7, 4.2.4.2", []string{"X-Forwarded-For"}, "1.3.3.7"},           // X-Forwarded-For with multiple entries (space+comma separator)
		{"127.0.0.1:9999", "X-Forwarded-For", "", []string{"X-Forwarded-For"}, "127.0.0.1"},                         // Empty header
		{"127.0.0.1:9999?ip=1.2.3.4", "", "", nil, "1.2.3.4"},                                                       // passed in "ip" parameter
		{"127.0.0.1:9999?ip=1.2.3.4", "X-Forwarded-For", "1.3.3.7,4.2.4.2", []string{"X-Forwarded-For"}, "1.2.3.4"}, // ip parameter wins over X-Forwarded-For with multiple entries

		{"127.0.0.1:9999", "X-Real-IP", "1.3.3.7:1337", []string{"X-Real-IP", "X-Forwarded-For"}, "1.3.3.7"},                       // Trusted header matches (with port)
		{"127.0.0.1:9999", "X-Forwarded-For", "1.3.3.7:1337", []string{"X-Real-IP", "X-Forwarded-For"}, "1.3.3.7"},                 // Second trusted header matches (with port)
		{"127.0.0.1:9999", "X-Forwarded-For", "1.3.3.7:1337,4.2.4.2:4242", []string{"X-Forwarded-For"}, "1.3.3.7"},                 // X-Forwarded-For with multiple entries (commas separator, with port)
		{"127.0.0.1:9999", "X-Forwarded-For", "1.3.3.7:1337, 4.2.4.2:4242", []string{"X-Forwarded-For"}, "1.3.3.7"},                // X-Forwarded-For with multiple entries (space+comma separator, with port)
		{"127.0.0.1:9999?ip=1.2.3.4:1234", "", "", nil, "1.2.3.4"},                                                                 // passed in "ip" parameter (with port)
		{"127.0.0.1:9999?ip=1.2.3.4:1234", "X-Forwarded-For", "1.3.3.7:1337,4.2.4.2:4242", []string{"X-Forwarded-For"}, "1.2.3.4"}, // ip parameter wins over X-Forwarded-For with multiple entries (with port)
	}
	testIpFromRequest(t, tests)
}
func TestIPv6FromRequest(t *testing.T) {
	var tests = []ipTestCase{
		{"[::1]:9999", "", "", nil, "::1"},                                                                                                  // No header given
		{"[::1]:9999", "X-Real-IP", "::ffff:103:307", nil, "::1"},                                                                           // Trusted header is empty
		{"[::1]:9999", "X-Real-IP", "::ffff:103:307", []string{"X-Foo-Bar"}, "::1"},                                                         // Trusted header does not match
		{"[::1]:9999", "X-Real-IP", "::ffff:103:307", []string{"X-Real-IP", "X-Forwarded-For"}, "::ffff:103:307"},                           // Trusted header matches
		{"[::1]:9999", "X-Forwarded-For", "::ffff:103:307", []string{"X-Real-IP", "X-Forwarded-For"}, "::ffff:103:307"},                     // Second trusted header matches
		{"[::1]:9999", "X-Forwarded-For", "::ffff:103:307,::ffff:402:402", []string{"X-Forwarded-For"}, "::ffff:103:307"},                   // X-Forwarded-For with multiple entries (commas separator)
		{"[::1]:9999", "X-Forwarded-For", "::ffff:103:307, ::ffff:402:402", []string{"X-Forwarded-For"}, "::ffff:103:307"},                  // X-Forwarded-For with multiple entries (space+comma separator)
		{"[::1]:9999", "X-Forwarded-For", "", []string{"X-Forwarded-For"}, "::1"},                                                           // Empty header
		{"[::1]:9999?ip=::ffff:102:304", "", "", nil, "::ffff:102:304"},                                                                     // passed in "ip" parameter
		{"[::1]:9999?ip=::ffff:102:304", "X-Forwarded-For", "::ffff:103:307,::ffff:402:402", []string{"X-Forwarded-For"}, "::ffff:102:304"}, // ip parameter wins over X-Forwarded-For with multiple entries

		{"[::1]:9999", "X-Real-IP", "[::ffff:103:307]:1337", []string{"X-Real-IP", "X-Forwarded-For"}, "::ffff:103:307"},                                         // Trusted header matches (with port)
		{"[::1]:9999", "X-Forwarded-For", "[::ffff:103:307]:1337", []string{"X-Real-IP", "X-Forwarded-For"}, "::ffff:103:307"},                                   // Second trusted header matches (with port)
		{"[::1]:9999", "X-Forwarded-For", "[::ffff:103:307]:1337,[::ffff:402:402]:4242", []string{"X-Forwarded-For"}, "::ffff:103:307"},                          // X-Forwarded-For with multiple entries (commas separator, with port)
		{"[::1]:9999", "X-Forwarded-For", "[::ffff:103:307]:1337, [::ffff:402:402]:4242", []string{"X-Forwarded-For"}, "::ffff:103:307"},                         // X-Forwarded-For with multiple entries (space+comma separator, with port)
		{"[::1]:9999?ip=[::ffff:102:304]:1234", "", "", nil, "::ffff:102:304"},                                                                                   // passed in "ip" parameter (with port)
		{"[::1]:9999?ip=[::ffff:102:304]:1234", "X-Forwarded-For", "[::ffff:103:307]:1337,[::ffff:402:402]:4242", []string{"X-Forwarded-For"}, "::ffff:102:304"}, // ip parameter wins over X-Forwarded-For with multiple entries (with port)
	}
	testIpFromRequest(t, tests)
}

func testIpFromRequest(t *testing.T, tests []ipTestCase) {
	for _, tt := range tests {
		u, err := url.Parse("http://" + tt.remoteAddr)
		if err != nil {
			t.Fatal(err)
		}
		r := &http.Request{
			RemoteAddr: u.Host,
			Header:     http.Header{},
			URL:        u,
		}
		r.Header.Add(tt.headerKey, tt.headerValue)
		ip, err := ipFromRequest(tt.trustedHeaders, r, true)
		if err != nil {
			t.Fatal(err)
		}
		out := net.ParseIP(tt.out)
		if !ip.Equal(out) {
			t.Errorf("Expected %s, got %s", out, ip)
		}
	}
}

func TestCLIMatcher(t *testing.T) {
	browserUserAgent := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_8_4) " +
		"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/30.0.1599.28 " +
		"Safari/537.36"
	var tests = []struct {
		in  string
		out bool
	}{
		{"curl/7.26.0", true},
		{"Wget/1.13.4 (linux-gnu)", true},
		{"Wget", true},
		{"fetch libfetch/2.0", true},
		{"HTTPie/0.9.3", true},
		{"httpie-go/0.6.0", true},
		{"Go 1.1 package http", true},
		{"Go-http-client/1.1", true},
		{"Go-http-client/2.0", true},
		{"ddclient/3.8.3", true},
		{"Mikrotik/6.x Fetch", true},
		{browserUserAgent, false},
	}
	for _, tt := range tests {
		r := &http.Request{Header: http.Header{"User-Agent": []string{tt.in}}}
		if got := cliMatcher(r); got != tt.out {
			t.Errorf("Expected %t, got %t for %q", tt.out, got, tt.in)
		}
	}
}

func TestGenerate204Handler(t *testing.T) {
	log.SetOutput(io.Discard)
	s := httptest.NewServer(testServer().Handler())
	resp, err := http.Get(s.URL + "/generate_204")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Expected 204, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if len(body) != 0 {
		t.Errorf("Expected empty body, got %q", body)
	}
}

func TestSpeedTestHandler(t *testing.T) {
	log.SetOutput(io.Discard)
	s := httptest.NewServer(testServer().Handler())

	var tests = []struct {
		path           string
		wantBytes      int64
		wantDisposName string
	}{
		{"/1mb", 1 * 1024 * 1024, "random-1mb.bin"},
		{"/10mb", 10 * 1024 * 1024, "random-10mb.bin"},
	}

	for _, tt := range tests {
		resp, err := http.Get(s.URL + tt.path)
		if err != nil {
			t.Fatalf("%s: %v", tt.path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s: expected status 200, got %d", tt.path, resp.StatusCode)
		}
		if got := resp.ContentLength; got != tt.wantBytes {
			t.Errorf("%s: expected Content-Length %d, got %d", tt.path, tt.wantBytes, got)
		}
		disp := resp.Header.Get("Content-Disposition")
		if disp != `attachment; filename="`+tt.wantDisposName+`"` {
			t.Errorf("%s: unexpected Content-Disposition: %q", tt.path, disp)
		}
	}
}

func TestSpeedTestHandlerInvalid(t *testing.T) {
	log.SetOutput(io.Discard)
	s := httptest.NewServer(testServer().Handler())

	var tests = []struct {
		path string
	}{
		{"/11gb"}, // exceeds 10 GB limit
		{"/0mb"},  // zero size
	}

	for _, tt := range tests {
		_, status, err := httpGet(s.URL+tt.path, jsonMediaType, "")
		if err != nil {
			t.Fatalf("%s: %v", tt.path, err)
		}
		if status != http.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d", tt.path, status)
		}
	}
}

func TestHeadHandler(t *testing.T) {
	log.SetOutput(io.Discard)
	s := httptest.NewServer(testServer().Handler())
	req, err := http.NewRequest(http.MethodHead, s.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Expected 204, got %d", resp.StatusCode)
	}
}

func TestLoggingMiddleware(t *testing.T) {
	log.SetOutput(io.Discard)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	ts := httptest.NewServer(loggingMiddleware(inner))
	defer ts.Close()
	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusTeapot {
		t.Errorf("Expected 418, got %d", resp.StatusCode)
	}
}

func TestLoggingMiddlewareOutput(t *testing.T) {
	var buf strings.Builder
	log.SetOutput(&buf)
	defer log.SetOutput(io.Discard)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	ts := httptest.NewServer(loggingMiddleware(inner))
	defer ts.Close()

	if _, err := http.Get(ts.URL + "/test-path"); err != nil {
		t.Fatal(err)
	}

	logged := buf.String()
	if !strings.Contains(logged, "GET") {
		t.Errorf("Expected method in log, got: %q", logged)
	}
	if !strings.Contains(logged, "/test-path") {
		t.Errorf("Expected path in log, got: %q", logged)
	}
	if !strings.Contains(logged, "200") {
		t.Errorf("Expected status code in log, got: %q", logged)
	}
}

func TestLoggingMiddlewareDefaultStatus(t *testing.T) {
	log.SetOutput(io.Discard)
	// Handler that writes body without calling WriteHeader — should default to 200
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	ts := httptest.NewServer(loggingMiddleware(inner))
	defer ts.Close()
	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestNew(t *testing.T) {
	r, _ := geo.Open("", "", "")
	cache := NewCache(0)
	s := New(r, cache, false)
	if s == nil {
		t.Fatal("Expected non-nil server")
	}
}

func TestInternalServerError(t *testing.T) {
	e := internalServerError(fmt.Errorf("boom"))
	if e.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500, got %d", e.Code)
	}
}

func TestInvalidIPQueryParam(t *testing.T) {
	log.SetOutput(io.Discard)
	s := httptest.NewServer(testServer().Handler())

	endpoints := []string{
		"/ip", "/country", "/country-iso", "/city", "/coordinates", "/asn", "/asn-org",
	}
	for _, ep := range endpoints {
		_, status, err := httpGet(s.URL+ep+"?ip=not-an-ip", "", "")
		if err != nil {
			t.Fatalf("%s: %v", ep, err)
		}
		if status != http.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d", ep, status)
		}
	}
	// JSON handler
	_, status, err := httpGet(s.URL+"/json?ip=not-an-ip", jsonMediaType, "")
	if err != nil {
		t.Fatal(err)
	}
	if status != http.StatusBadRequest {
		t.Errorf("/json: expected 400, got %d", status)
	}
}

func TestCacheResizeHandlerInvalidBody(t *testing.T) {
	log.SetOutput(io.Discard)
	srv := testServer()
	srv.profile = true
	s := httptest.NewServer(srv.Handler())

	// Non-numeric body
	_, body, err := httpPost(s.URL+"/debug/cache/resize", "abc")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "error") {
		t.Errorf("Expected error in response, got %q", body)
	}

	// Negative capacity (triggers Resize error)
	_, body, err = httpPost(s.URL+"/debug/cache/resize", "-1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "error") {
		t.Errorf("Expected error for negative capacity, got %q", body)
	}
}

func TestWrapHandlerFunc(t *testing.T) {
	log.SetOutput(io.Discard)
	srv := testServer()
	srv.profile = true
	s := httptest.NewServer(srv.Handler())
	_, status, err := httpGet(s.URL+"/debug/pprof/cmdline", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if status != http.StatusOK {
		t.Errorf("Expected 200, got %d", status)
	}
}

func TestServeHTTP500(t *testing.T) {
	log.SetOutput(io.Discard)
	h := appHandler(func(w http.ResponseWriter, r *http.Request) *appError {
		return internalServerError(fmt.Errorf("server error"))
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500, got %d", rec.Code)
	}
}

func TestIPFromRequestSplitHostPortError(t *testing.T) {
	// "::not:valid::" has colons but SplitHostPort fails → error
	r := &http.Request{
		RemoteAddr: "::not:valid::",
		Header:     http.Header{},
		URL:        &url.URL{},
	}
	_, err := ipFromRequest(nil, r, false)
	if err == nil {
		t.Error("Expected error for invalid RemoteAddr, got nil")
	}
}

func TestDefaultHandler(t *testing.T) {
	log.SetOutput(io.Discard)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(`{{.IP}}`), 0644); err != nil {
		t.Fatal(err)
	}
	srv := testServer()
	srv.Template = dir
	s := httptest.NewServer(srv.Handler())
	_, status, err := httpGet(s.URL, "", "Mozilla/5.0")
	if err != nil {
		t.Fatal(err)
	}
	if status != http.StatusOK {
		t.Errorf("Expected 200, got %d", status)
	}
}
