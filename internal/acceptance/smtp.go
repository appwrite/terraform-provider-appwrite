package acceptance

import (
	"bufio"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// SMTPServer starts a stub SMTP server in the test process and returns the host
// and port Appwrite should be configured with to reach it.
//
// Appwrite connects to the SMTP server and authenticates every time custom
// SMTP is saved enabled, and rejects the configuration when that fails. Custom
// email templates can only be saved while custom SMTP is enabled. So testing
// either needs a server Appwrite can actually reach, and a dummy host name is
// not enough.
//
// The stub greets, offers AUTH PLAIN and LOGIN, accepts any credentials and
// says goodbye. That is all Appwrite does on save. It does not offer STARTTLS,
// so a configuration with secure set cannot be enabled against it.
//
// The stub listens on every interface, and APPWRITE_TEST_SMTP_HOST names the
// address Appwrite reaches this machine at: host.docker.internal on Docker
// Desktop, or the Docker network's gateway on Linux. The CI bootstrap script
// exports it. Without it the test skips.
func SMTPServer(t *testing.T) (host string, port int) {
	t.Helper()

	// Fixtures run before resource.Test, so they sit outside the TF_ACC gate.
	if os.Getenv(resource.EnvTfAcc) == "" {
		t.Skipf("acceptance tests skipped unless %s is set", resource.EnvTfAcc)
	}
	host = os.Getenv("APPWRITE_TEST_SMTP_HOST")
	if host == "" {
		t.Skip("APPWRITE_TEST_SMTP_HOST must be set to the address Appwrite reaches this machine at, such as host.docker.internal")
	}

	var config net.ListenConfig
	listener, err := config.Listen(t.Context(), "tcp", ":0")
	if err != nil {
		t.Fatalf("starting stub SMTP server: %v", err)
	}
	address, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		_ = listener.Close()
		t.Fatalf("stub SMTP server is listening on %v, which is not a TCP address", listener.Addr())
	}

	var wg sync.WaitGroup
	t.Cleanup(func() {
		_ = listener.Close()
		wg.Wait()
	})

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				serveSMTP(conn)
			}()
		}
	}()

	return host, address.Port
}

func serveSMTP(conn net.Conn) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))

	reader := bufio.NewReader(conn)
	reply := func(lines ...string) bool {
		_, err := conn.Write([]byte(strings.Join(lines, "\r\n") + "\r\n"))
		return err == nil
	}

	if !reply("220 stub ESMTP") {
		return
	}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		command := strings.ToUpper(strings.TrimSpace(line))
		var ok bool
		switch {
		case strings.HasPrefix(command, "EHLO"):
			ok = reply("250-stub", "250 AUTH PLAIN LOGIN")
		case strings.HasPrefix(command, "AUTH"):
			ok = reply("235 2.7.0 Authentication successful")
		case strings.HasPrefix(command, "QUIT"):
			reply("221 2.0.0 Bye")
			return
		default:
			ok = reply("250 OK")
		}
		if !ok {
			return
		}
	}
}
