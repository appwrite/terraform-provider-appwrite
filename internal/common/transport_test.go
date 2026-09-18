package common

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/appwrite/sdk-for-go/v7/client"
)

// roundTripperFunc adapts a function to http.RoundTripper.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func response(status int, header http.Header) *http.Response {
	if header == nil {
		header = http.Header{}
	}
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     header,
		Body:       io.NopCloser(strings.NewReader("{}")),
	}
}

func TestRetryTransportRetriesRetryableStatuses(t *testing.T) {
	t.Parallel()

	for _, status := range []int{
		http.StatusTooManyRequests,
		http.StatusRequestTimeout,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout,
	} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			t.Parallel()

			var calls atomic.Int32
			rt := &retryTransport{
				maxRetries: 2,
				next: roundTripperFunc(func(*http.Request) (*http.Response, error) {
					if calls.Add(1) < 3 {
						// Retry-After keeps the test fast and also exercises the
						// header path rather than the backoff.
						return response(status, http.Header{"Retry-After": []string{"0"}}), nil
					}
					return response(http.StatusOK, nil), nil
				}),
			}

			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.test/v1/health", nil)
			if err != nil {
				t.Fatal(err)
			}
			resp, err := rt.RoundTrip(req)
			if err != nil {
				t.Fatalf("RoundTrip: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("status = %d, want 200 after retrying", resp.StatusCode)
			}
			if got := calls.Load(); got != 3 {
				t.Errorf("attempts = %d, want 3", got)
			}
		})
	}
}

func TestRetryTransportDoesNotRetryClientErrors(t *testing.T) {
	t.Parallel()

	// A 404 or a 401 will return the same answer however many times it is sent,
	// and retrying a 409 would paper over a genuine conflict.
	for _, status := range []int{
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusNotImplemented,
	} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			t.Parallel()

			var calls atomic.Int32
			rt := &retryTransport{
				maxRetries: 5,
				next: roundTripperFunc(func(*http.Request) (*http.Response, error) {
					calls.Add(1)
					return response(status, nil), nil
				}),
			}

			req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.test/v1/x", nil)
			resp, err := rt.RoundTrip(req)
			if err != nil {
				t.Fatalf("RoundTrip: %v", err)
			}
			defer resp.Body.Close()

			if got := calls.Load(); got != 1 {
				t.Errorf("attempts = %d, want 1; %d must not be retried", got, status)
			}
		})
	}
}

func TestRetryTransportStopsAtMaxRetries(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	rt := &retryTransport{
		maxRetries: 3,
		next: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return response(http.StatusServiceUnavailable, http.Header{"Retry-After": []string{"0"}}), nil
		}),
	}

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.test/v1/x", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()

	// One initial attempt plus three retries.
	if got := calls.Load(); got != 4 {
		t.Errorf("attempts = %d, want 4 (1 initial + 3 retries)", got)
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want the last failure surfaced to the caller", resp.StatusCode)
	}
}

// A request whose body cannot be rewound must be sent exactly once. Replaying it
// would send an empty body, and the server would then fail it for a reason
// unrelated to the original error -- which is far harder to diagnose than the
// rate limit that started it.
func TestRetryTransportDoesNotReplayUnrewindableBody(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	rt := &retryTransport{
		maxRetries: 3,
		next: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return response(http.StatusTooManyRequests, http.Header{"Retry-After": []string{"0"}}), nil
		}),
	}

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://example.test/v1/x",
		io.NopCloser(strings.NewReader(`{"name":"x"}`)))
	req.GetBody = nil // what a streaming or unknown-length body looks like

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()

	if got := calls.Load(); got != 1 {
		t.Errorf("attempts = %d, want 1 for a body that cannot be rewound", got)
	}
}

func TestRetryTransportReplaysRewindableBody(t *testing.T) {
	t.Parallel()

	var bodies []string
	var calls atomic.Int32
	rt := &retryTransport{
		maxRetries: 2,
		next: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			b, _ := io.ReadAll(r.Body)
			bodies = append(bodies, string(b))
			if calls.Add(1) < 2 {
				return response(http.StatusTooManyRequests, http.Header{"Retry-After": []string{"0"}}), nil
			}
			return response(http.StatusOK, nil), nil
		}),
	}

	// http.NewRequest sets GetBody for a strings.Reader, which is what the SDK
	// produces for a JSON body.
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://example.test/v1/x",
		strings.NewReader(`{"name":"x"}`))
	if err != nil {
		t.Fatal(err)
	}

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()

	if len(bodies) != 2 {
		t.Fatalf("attempts = %d, want 2", len(bodies))
	}
	if bodies[0] != bodies[1] {
		t.Errorf("retry sent a different body: %q then %q", bodies[0], bodies[1])
	}
}

func TestRetryTransportRetriesTransportErrorsButNotCancellation(t *testing.T) {
	t.Parallel()

	t.Run("connection error is retried", func(t *testing.T) {
		t.Parallel()
		var calls atomic.Int32
		rt := &retryTransport{
			maxRetries: 2,
			next: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				if calls.Add(1) < 3 {
					return nil, errors.New("connection reset by peer")
				}
				return response(http.StatusOK, nil), nil
			}),
		}
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.test/v1/x", nil)
		resp, err := rt.RoundTrip(req)
		if err != nil {
			t.Fatalf("RoundTrip: %v", err)
		}
		defer resp.Body.Close()
		if got := calls.Load(); got != 3 {
			t.Errorf("attempts = %d, want 3", got)
		}
	})

	t.Run("cancellation is not retried", func(t *testing.T) {
		t.Parallel()
		var calls atomic.Int32
		rt := &retryTransport{
			maxRetries: 5,
			next: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return nil, context.Canceled
			}),
		}
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.test/v1/x", nil)
		resp, err := rt.RoundTrip(req)
		if resp != nil {
			resp.Body.Close()
		}
		if err == nil {
			t.Fatal("expected the cancellation to be returned")
		}
		if got := calls.Load(); got != 1 {
			t.Errorf("attempts = %d, want 1; the caller already gave up", got)
		}
	})
}

// A canceled context must abandon the wait rather than sleep through it, and
// must report the cancellation rather than the stale response that triggered the
// retry -- reporting a 429 for a timeout sends the reader after rate limits.
func TestRetryTransportHonorsContextDuringBackoff(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	rt := &retryTransport{
		maxRetries: 5,
		next: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			cancel()
			return response(http.StatusTooManyRequests, http.Header{"Retry-After": []string{"30"}}), nil
		}),
	}

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://example.test/v1/x", nil)

	done := make(chan struct{})
	var err error
	go func() {
		resp, rtErr := rt.RoundTrip(req)
		if resp != nil {
			resp.Body.Close()
		}
		err = rtErr
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("RoundTrip slept through a canceled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func TestParseRetryAfter(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		header   string
		wantOK   bool
		wantZero bool
	}{
		"seconds":            {header: "5", wantOK: true},
		"zero seconds":       {header: "0", wantOK: true, wantZero: true},
		"negative rejected":  {header: "-1", wantOK: false},
		"empty":              {header: "", wantOK: false},
		"garbage":            {header: "soon", wantOK: false},
		"http date in past":  {header: "Mon, 02 Jan 2006 15:04:05 GMT", wantOK: true, wantZero: true},
		"absurdly large cap": {header: "999999", wantOK: true},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, ok := parseRetryAfter(tc.header)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if !ok {
				return
			}
			if tc.wantZero && got != 0 {
				t.Errorf("duration = %v, want 0", got)
			}
			if got > maxRetryBackoff {
				t.Errorf("duration = %v, want it capped at %v", got, maxRetryBackoff)
			}
		})
	}
}

func TestBackoffIsCappedAndJittered(t *testing.T) {
	t.Parallel()

	// Jitter is what stops parallel resources that hit the same rate limit from
	// waking together and hitting it again, so identical waits are a bug.
	seen := make(map[time.Duration]bool)
	for range 50 {
		d := backoff(4)
		if d <= 0 {
			t.Fatalf("backoff returned %v, want a positive wait", d)
		}
		if d > maxRetryBackoff {
			t.Fatalf("backoff returned %v, want at most %v", d, maxRetryBackoff)
		}
		seen[d] = true
	}
	if len(seen) < 2 {
		t.Error("backoff returned the same wait every time; the jitter is not working")
	}

	// A high attempt number must not overflow into a negative or enormous wait.
	for _, attempt := range []int{10, 32, 63, 100} {
		if d := backoff(attempt); d <= 0 || d > maxRetryBackoff {
			t.Errorf("backoff(%d) = %v, want a positive wait at most %v", attempt, d, maxRetryBackoff)
		}
	}
}

// The provider logs every request at debug level, and debug output routinely
// ends up pasted into bug reports. An API key appearing there is a credential
// leak, so this is the most important test in the file.
func TestSafeHeadersRedactsCredentials(t *testing.T) {
	t.Parallel()

	header := http.Header{
		"X-Appwrite-Key":     []string{"standard_deadbeefsecret"},
		"x-appwrite-jwt":     []string{"ey.jwt.secret"},
		"X-APPWRITE-SESSION": []string{"session-secret"},
		"X-Appwrite-Dev-Key": []string{"dev-secret"},
		"Authorization":      []string{"Bearer token-secret"},
		"Cookie":             []string{"a_session=secret"},
		"Set-Cookie":         []string{"a_session=secret"},
		"X-Appwrite-Project": []string{"my-project"},
		"Content-Type":       []string{"application/json"},
	}

	got := safeHeaders(header)

	for name, value := range got {
		if strings.Contains(value, "secret") {
			t.Errorf("header %q leaked its value: %q", name, value)
		}
	}

	// Redaction has to be casing-insensitive, because the SDK sets header names
	// in lower case and net/http canonicalizes them.
	for _, name := range []string{"X-Appwrite-Key", "x-appwrite-jwt", "X-APPWRITE-SESSION", "X-Appwrite-Dev-Key", "Authorization", "Cookie", "Set-Cookie"} {
		if got[name] != "(redacted)" {
			t.Errorf("header %q = %q, want (redacted)", name, got[name])
		}
	}

	// Non-secret headers stay legible, or the logs are useless for debugging.
	if got["X-Appwrite-Project"] != "my-project" {
		t.Errorf("project header = %q, want it preserved", got["X-Appwrite-Project"])
	}
	if got["Content-Type"] != "application/json" {
		t.Errorf("content type = %q, want it preserved", got["Content-Type"])
	}
}

func TestWithHTTPTransportLeavesSelfSignedToTheTransport(t *testing.T) {
	t.Parallel()

	// The SDK's own self-signed handling type-asserts the transport to
	// *http.Transport and fails the request on anything else. Since this chain
	// is not one, client.SelfSigned must stay false or every call breaks.
	clt := newTestClient()
	if err := WithHTTPTransport(HTTPConfig{Timeout: time.Second, SelfSigned: true, MaxRetries: 1})(&clt); err != nil {
		t.Fatalf("WithHTTPTransport: %v", err)
	}
	if clt.SelfSigned {
		t.Error("SelfSigned was left true; the SDK would then reject the wrapped transport")
	}
	if clt.Client == nil || clt.Client.Transport == nil {
		t.Fatal("no transport was installed")
	}
	if _, ok := clt.Client.Transport.(*retryTransport); !ok {
		t.Errorf("outermost transport is %T, want the retry transport so each attempt is logged", clt.Client.Transport)
	}
}

// End to end against a real server, so the chain is exercised as it ships:
// self-signed TLS accepted, a rate limit retried, the response returned.
func TestTransportChainAgainstSelfSignedServer(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	clt := newTestClient()
	if err := WithHTTPTransport(HTTPConfig{Timeout: 10 * time.Second, SelfSigned: true, MaxRetries: 3})(&clt); err != nil {
		t.Fatalf("WithHTTPTransport: %v", err)
	}

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := clt.Client.Do(req)
	if err != nil {
		t.Fatalf("request through the chain failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200 after the rate limit was retried", resp.StatusCode)
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("server saw %d requests, want 2", got)
	}
}

func TestTransportChainRejectsUntrustedTLSWithoutSelfSigned(t *testing.T) {
	t.Parallel()

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	clt := newTestClient()
	// self_signed off: the server's certificate is untrusted and must be refused.
	// Verifying this is how we know the flag still means something.
	if err := WithHTTPTransport(HTTPConfig{Timeout: 10 * time.Second, SelfSigned: false, MaxRetries: 0})(&clt); err != nil {
		t.Fatalf("WithHTTPTransport: %v", err)
	}

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := clt.Client.Do(req)
	if err == nil {
		resp.Body.Close()
		t.Fatal("an untrusted certificate was accepted with self_signed disabled")
	}
}

func TestAppendedUserAgent(t *testing.T) {
	t.Setenv("TF_APPEND_USER_AGENT", "  terraform-cloud/1.2.3  ")
	if got := AppendedUserAgent(); got != "terraform-cloud/1.2.3" {
		t.Errorf("AppendedUserAgent() = %q, want it trimmed", got)
	}

	t.Setenv("TF_APPEND_USER_AGENT", "")
	if got := AppendedUserAgent(); got != "" {
		t.Errorf("AppendedUserAgent() = %q, want empty", got)
	}
}

func TestWithUserAgentAppendsEnvironment(t *testing.T) {
	t.Setenv("TF_APPEND_USER_AGENT", "terragrunt/0.55.0")

	clt := newTestClient()
	if err := WithUserAgent("2.1.0")(&clt); err != nil {
		t.Fatalf("WithUserAgent: %v", err)
	}
	got := clt.Headers["user-agent"]
	if !strings.HasPrefix(got, "terraform-provider-appwrite/2.1.0") {
		t.Errorf("user-agent = %q, want it to start with the provider and version", got)
	}
	if !strings.Contains(got, "terragrunt/0.55.0") {
		t.Errorf("user-agent = %q, want TF_APPEND_USER_AGENT appended", got)
	}
}

// newTestClient returns an SDK client shaped the way appwrite.NewClient leaves
// one: maps initialized, no transport yet.
func newTestClient() client.Client {
	return client.Client{
		Headers: map[string]string{},
		Config:  map[string]string{},
	}
}
