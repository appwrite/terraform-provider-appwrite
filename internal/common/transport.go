package common

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/appwrite/sdk-for-go/v7/client"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// DefaultMaxRetries is the number of times a retryable request is tried again.
//
// Lower than the AWS provider's 25 on purpose. That number is calibrated for a
// service with generous per-account limits and long-lived eventual consistency;
// Appwrite Cloud's rate limits are tighter and its errors are mostly immediate,
// so a long retry budget mainly converts one slow failure into several. Five
// covers a rate-limit window and a restarting sidecar without making a genuine
// outage take minutes to report.
const DefaultMaxRetries = 5

// maxRetryBackoff caps a single wait. Without it the sixth attempt's exponential
// backoff would dwarf the operation it is protecting.
const maxRetryBackoff = 30 * time.Second

// HTTPConfig describes the transport the provider builds for every SDK client.
type HTTPConfig struct {
	Timeout    time.Duration
	SelfSigned bool
	MaxRetries int
	UserAgent  string
}

// WithHTTPTransport installs the provider's transport chain on an SDK client.
//
// It must be applied after appwrite.WithTimeout, which replaces the whole
// http.Client and would otherwise discard this.
//
// Self-signed certificates are handled here rather than through
// appwrite.WithSelfSigned. The SDK's implementation type-asserts the transport
// to *http.Transport and fails the request outright on anything else, so a
// provider that wraps the transport -- for retries, for logging -- cannot also
// use the SDK's flag. Setting InsecureSkipVerify on the base transport in this
// chain is the same effect with none of the incompatibility, and it leaves
// client.SelfSigned false so the SDK's assertion is never reached.
func WithHTTPTransport(cfg HTTPConfig) client.ClientOption {
	return func(clt *client.Client) error {
		base, ok := http.DefaultTransport.(*http.Transport)
		if !ok {
			return fmt.Errorf("unexpected default HTTP transport %T", http.DefaultTransport)
		}
		transport := base.Clone()
		if cfg.SelfSigned {
			if transport.TLSClientConfig == nil {
				transport.TLSClientConfig = &tls.Config{} //nolint:gosec // InsecureSkipVerify is the point of self_signed.
			}
			transport.TLSClientConfig.InsecureSkipVerify = true
		}

		// Logging sits inside retrying so that each attempt is logged
		// separately. The other order logs one request and hides the fact that
		// it was made four times.
		var chain http.RoundTripper = &loggingTransport{next: transport}
		chain = &retryTransport{next: chain, maxRetries: cfg.MaxRetries}

		if clt.Client == nil {
			httpClient, err := client.GetDefaultClient(cfg.Timeout)
			if err != nil {
				return err
			}
			clt.Client = httpClient
		}
		clt.Client.Transport = chain

		// Left false deliberately: the TLS config above already covers it, and
		// setting it would send the SDK down the path that rejects this chain.
		clt.SelfSigned = false
		return nil
	}
}

// retryTransport retries requests that failed for a reason likely to pass.
type retryTransport struct {
	next       http.RoundTripper
	maxRetries int
}

func (r *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Only a negative value falls back to the default. Zero means zero: it is
	// the documented way to disable retrying, and treating it as "unset" made
	// that opt-out do the opposite of what it says.
	attempts := r.maxRetries
	if attempts < 0 {
		attempts = DefaultMaxRetries
	}

	// A request whose body cannot be rewound can only be sent once. Retrying it
	// would replay an empty body, and the server would reject it for a reason
	// that has nothing to do with the original failure.
	replayable := req.Body == nil || req.Body == http.NoBody || req.GetBody != nil

	ctx := req.Context()

	for attempt := 0; ; attempt++ {
		if attempt > 0 {
			if err := rewind(req); err != nil {
				return nil, err
			}
		}

		resp, err := r.next.RoundTrip(req)

		if !replayable || attempt >= attempts {
			return resp, err
		}
		wait, ok := retryAfter(req.Method, resp, err, attempt)
		if !ok {
			return resp, err
		}

		// A response that is going to be retried must be drained and closed, or
		// its connection is not returned to the pool.
		if resp != nil {
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
			resp.Body.Close()
		}

		tflog.Debug(ctx, "retrying Appwrite request", map[string]any{
			"method":  req.Method,
			"url":     req.URL.Path,
			"attempt": attempt + 1,
			"of":      attempts,
			"in":      wait.String(),
			"reason":  retryReason(resp, err),
		})

		select {
		case <-ctx.Done():
			// Returning the context error rather than the last response: the
			// caller's deadline expired, and reporting a stale 429 for that
			// would send whoever reads it looking at rate limits.
			return nil, ctx.Err()
		case <-time.After(wait):
		}
	}
}

func rewind(req *http.Request) error {
	if req.GetBody == nil {
		return nil
	}
	body, err := req.GetBody()
	if err != nil {
		return fmt.Errorf("could not rewind request body for retry: %w", err)
	}
	req.Body = body
	return nil
}

// idempotentMethods are the methods it is safe to send twice.
//
// PATCH and POST are absent deliberately. Appwrite creates resources with POST
// and updates them with PATCH, and neither carries an idempotency key, so a
// second send is a second create or a second mutation.
var idempotentMethods = map[string]bool{
	http.MethodGet:     true,
	http.MethodHead:    true,
	http.MethodOptions: true,
	http.MethodTrace:   true,
	http.MethodPut:     true,
	http.MethodDelete:  true,
}

// retryAfter reports how long to wait before the next attempt, and whether to
// make one at all.
//
// What matters is not which status code came back but whether the server might
// already have acted. A 429 is a refusal: the request was rejected at the edge
// and nothing happened, so replaying it is safe whatever the method. Everything
// else here is ambiguous -- a connection that died mid-flight, or a 500 from a
// container that fell over after committing -- and replaying a POST in that
// situation creates a second deployment that Terraform holds no state for.
// Ambiguous failures are therefore retried only for methods that can be sent
// twice without consequence.
//
// The cost of the asymmetry is that a create interrupted by a genuine blip now
// reports the error instead of quietly succeeding on the second try. That is the
// right trade: a failed apply the operator can rerun is recoverable, and
// orphaned billable infrastructure nobody knows about is not.
func retryAfter(method string, resp *http.Response, err error, attempt int) (time.Duration, bool) {
	ambiguousRetryAllowed := idempotentMethods[strings.ToUpper(method)]

	if err != nil {
		// A transport-level error produced no response at all: a reset, a DNS
		// blip, a TLS handshake failure. Whether the server saw the request is
		// unknowable from here. A canceled context is never retried, since the
		// caller has already given up.
		if isContextError(err) || !ambiguousRetryAllowed {
			return 0, false
		}
		return backoff(attempt), true
	}
	if resp == nil {
		return 0, false
	}

	switch resp.StatusCode {
	case http.StatusTooManyRequests:
		// Appwrite sends Retry-After on rate limits. Honoring it is both more
		// polite and faster than guessing, and ignoring it is how a client
		// turns one rate limit into a longer one.
		if wait, ok := parseRetryAfter(resp.Header.Get("Retry-After")); ok {
			return wait, true
		}
		return backoff(attempt), true
	case http.StatusRequestTimeout,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout,
		// Appwrite returns 500 for genuine server faults, which a retry can
		// clear when the cause is a restarting container. Deliberately the only
		// 5xx treated this way beyond the gateway codes, since a 501 or 505 will
		// never succeed however many times it is sent.
		http.StatusInternalServerError:
		if !ambiguousRetryAllowed {
			return 0, false
		}
		return backoff(attempt), true
	default:
		return 0, false
	}
}

func retryReason(resp *http.Response, err error) string {
	if err != nil {
		return err.Error()
	}
	if resp != nil {
		return resp.Status
	}
	return "unknown"
}

func isContextError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// backoff returns an exponentially increasing wait with full jitter.
//
// Jitter is not decoration. Terraform applies resources in parallel by default,
// so without it every goroutine that hit the same rate limit wakes at the same
// moment and hits it again together.
func backoff(attempt int) time.Duration {
	base := maxRetryBackoff
	// Computed by shifting rather than math.Pow, and guarded. Converting an
	// overflowed float to a Duration wraps negative, and the cap comparison then
	// passes, leaving a negative bound that makes the jitter call panic. A high
	// attempt number should not be reachable through max_retries, but "should not
	// be reachable" is not a reason for a panic to be possible.
	if attempt >= 0 && attempt < 30 {
		if shifted := time.Second << uint(attempt); shifted > 0 && shifted < maxRetryBackoff {
			base = shifted
		}
	}
	// Full jitter: uniform over (0, base]. Keeps the expected wait at half the
	// nominal backoff while removing the synchronized retry.
	return time.Duration(rand.Int64N(int64(base))) + time.Millisecond
}

// parseRetryAfter reads both forms the header is allowed to take: a number of
// seconds, or an HTTP date.
func parseRetryAfter(value string) (time.Duration, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	if secs, err := strconv.Atoi(value); err == nil {
		if secs < 0 {
			return 0, false
		}
		wait := time.Duration(secs) * time.Second
		if wait > maxRetryBackoff {
			wait = maxRetryBackoff
		}
		return wait, true
	}
	if when, err := http.ParseTime(value); err == nil {
		wait := time.Until(when)
		if wait <= 0 {
			return 0, true
		}
		if wait > maxRetryBackoff {
			wait = maxRetryBackoff
		}
		return wait, true
	}
	return 0, false
}

// loggingTransport writes each request and response to the Terraform log at
// debug level, with credentials masked.
type loggingTransport struct {
	next http.RoundTripper
}

// sensitiveHeaders are never logged, in any casing.
//
// x-appwrite-key is an API key with full project authority, and the others are
// session credentials. A provider that logs them turns TF_LOG=debug into a
// credential leak, and debug output routinely ends up in bug reports.
var sensitiveHeaders = map[string]bool{
	"x-appwrite-key":          true,
	"x-appwrite-jwt":          true,
	"x-appwrite-session":      true,
	"x-appwrite-dev-key":      true,
	"authorization":           true,
	"cookie":                  true,
	"set-cookie":              true,
	"x-appwrite-organization": false,
}

func (l *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	started := time.Now()

	tflog.Debug(ctx, "Appwrite API request", map[string]any{
		"method":  req.Method,
		"url":     req.URL.String(),
		"headers": safeHeaders(req.Header),
	})

	resp, err := l.next.RoundTrip(req)
	elapsed := time.Since(started)

	if err != nil {
		tflog.Debug(ctx, "Appwrite API request failed", map[string]any{
			"method":  req.Method,
			"url":     req.URL.String(),
			"error":   err.Error(),
			"elapsed": elapsed.String(),
		})
		return resp, err
	}

	tflog.Debug(ctx, "Appwrite API response", map[string]any{
		"method":  req.Method,
		"url":     req.URL.String(),
		"status":  resp.StatusCode,
		"elapsed": elapsed.String(),
		"headers": safeHeaders(resp.Header),
	})
	return resp, nil
}

func safeHeaders(h http.Header) map[string]string {
	out := make(map[string]string, len(h))
	for name, values := range h {
		if sensitiveHeaders[strings.ToLower(name)] {
			out[name] = "(redacted)"
			continue
		}
		out[name] = strings.Join(values, ", ")
	}
	return out
}

// AppendedUserAgent returns the value of TF_APPEND_USER_AGENT.
//
// terraform-plugin-sdk honors this variable for free; a framework-only provider
// has to do it itself, which is why it currently has no effect here. It is how
// Terraform Cloud, Terragrunt and in-house wrappers identify themselves, and
// without it their traffic is indistinguishable from a developer's laptop in
// Appwrite's logs.
func AppendedUserAgent() string {
	return strings.TrimSpace(os.Getenv("TF_APPEND_USER_AGENT"))
}
