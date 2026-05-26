// internal/checker/dial.go
package checker

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"strings"
	"time"
)

// target holds the parsed components of a URL or bare hostname.
type target struct {
	Original string
	Scheme   string
	Host     string // hostname only, no port
	Port     string // port as string
	DialAddr string // "host:port" ready for net.Dial
}

// parseTarget parses any supported URL format into a target.
// Supports bare hostnames, host:port, and URI schemes.
func parseTarget(raw string) target {
	t := target{Original: raw}

	s := raw

	// extract scheme
	if idx := strings.Index(s, "://"); idx != -1 {
		t.Scheme = strings.ToLower(s[:idx])
		s = s[idx+3:]
	}

	// strip path, query, fragment
	if idx := strings.IndexAny(s, "/?#"); idx != -1 {
		s = s[:idx]
	}

	// split host and port
	host, port, err := net.SplitHostPort(s)
	if err != nil {
		// no port present — use scheme default
		t.Host = s
		t.Port = schemeDefaultPort(t.Scheme)
	} else {
		t.Host = host
		t.Port = port
	}

	t.DialAddr = net.JoinHostPort(t.Host, t.Port)
	return t
}

// schemeDefaultPort returns the default port for a given URI scheme.
func schemeDefaultPort(scheme string) string {
	switch scheme {
	case "https", "":
		return "443"
	case "smtps":
		return "465"
	case "smtp", "submission":
		return "587"
	case "imaps":
		return "993"
	case "imap":
		return "143"
	case "pop3s":
		return "995"
	case "pop3":
		return "110"
	case "ldaps":
		return "636"
	case "ldap":
		return "389"
	case "ftps":
		return "990"
	default:
		return "443"
	}
}

// isSTARTTLS reports whether a scheme negotiates TLS over a plain connection.
func isSTARTTLS(scheme string) bool {
	switch scheme {
	case "smtp", "submission", "imap", "pop3", "ldap", "ftp":
		return true
	}
	return false
}

// dialCerts connects to t and returns the peer certificate chain.
// Dispatches to plain TLS or the appropriate STARTTLS handler.
func dialCerts(t target, timeout time.Duration) ([]*x509.Certificate, error) {
	if isSTARTTLS(t.Scheme) {
		switch t.Scheme {
		case "smtp", "submission":
			return starttlsSMTP(t.DialAddr, t.Host, timeout)
		case "imap":
			return starttlsIMAP(t.DialAddr, t.Host, timeout)
		case "pop3":
			return starttlsPOP3(t.DialAddr, t.Host, timeout)
		default:
			return nil, fmt.Errorf("STARTTLS not implemented for scheme %q", t.Scheme)
		}
	}
	return dialTLS(t.DialAddr, t.Host, timeout)
}

// dialTLS performs a direct TLS handshake and returns the peer certificates.
func dialTLS(dialAddr, hostname string, timeout time.Duration) ([]*x509.Certificate, error) {
	conn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: timeout},
		"tcp", dialAddr,
		&tls.Config{ServerName: hostname},
	)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return nil, fmt.Errorf("no certificates received")
	}
	return certs, nil
}

// starttlsSMTP upgrades a plain SMTP connection to TLS via STARTTLS.
func starttlsSMTP(dialAddr, hostname string, timeout time.Duration) ([]*x509.Certificate, error) {
	conn, err := net.DialTimeout("tcp", dialAddr, timeout)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(timeout))

	r := bufio.NewReader(conn)

	// read server greeting: "220 mail.example.com ESMTP"
	if _, err := r.ReadString('\n'); err != nil {
		return nil, fmt.Errorf("SMTP greeting: %w", err)
	}

	// EHLO
	fmt.Fprintf(conn, "EHLO cert-checker\r\n")
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("SMTP EHLO: %w", err)
		}
		// multi-line response ends when 4th char is space, not dash
		if len(line) >= 4 && line[3] == ' ' {
			break
		}
	}

	// request STARTTLS
	fmt.Fprintf(conn, "STARTTLS\r\n")
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("SMTP STARTTLS: %w", err)
	}
	if !strings.HasPrefix(line, "220") {
		return nil, fmt.Errorf("SMTP STARTTLS not accepted: %s", strings.TrimSpace(line))
	}

	return upgradeTLS(conn, r, hostname, timeout)
}

// starttlsIMAP upgrades a plain IMAP connection to TLS via STARTTLS.
func starttlsIMAP(dialAddr, hostname string, timeout time.Duration) ([]*x509.Certificate, error) {
	conn, err := net.DialTimeout("tcp", dialAddr, timeout)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(timeout))

	r := bufio.NewReader(conn)

	// read server greeting: "* OK IMAP server ready"
	if _, err := r.ReadString('\n'); err != nil {
		return nil, fmt.Errorf("IMAP greeting: %w", err)
	}

	fmt.Fprintf(conn, "A001 STARTTLS\r\n")
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("IMAP STARTTLS: %w", err)
	}
	if !strings.Contains(line, "OK") {
		return nil, fmt.Errorf("IMAP STARTTLS not accepted: %s", strings.TrimSpace(line))
	}

	return upgradeTLS(conn, r, hostname, timeout)
}

// starttlsPOP3 upgrades a plain POP3 connection to TLS via STLS.
func starttlsPOP3(dialAddr, hostname string, timeout time.Duration) ([]*x509.Certificate, error) {
	conn, err := net.DialTimeout("tcp", dialAddr, timeout)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(timeout))

	r := bufio.NewReader(conn)

	// read server greeting: "+OK POP3 server ready"
	if _, err := r.ReadString('\n'); err != nil {
		return nil, fmt.Errorf("POP3 greeting: %w", err)
	}

	fmt.Fprintf(conn, "STLS\r\n")
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("POP3 STLS: %w", err)
	}
	if !strings.HasPrefix(line, "+OK") {
		return nil, fmt.Errorf("POP3 STLS not accepted: %s", strings.TrimSpace(line))
	}

	return upgradeTLS(conn, r, hostname, timeout)
}

// upgradeTLS performs the TLS handshake over an existing plain connection.
// The bufio.Reader is passed so any buffered data is not lost during upgrade.
func upgradeTLS(conn net.Conn, _ *bufio.Reader, hostname string, timeout time.Duration) ([]*x509.Certificate, error) {
	tlsConn := tls.Client(conn, &tls.Config{ServerName: hostname})
	tlsConn.SetDeadline(time.Now().Add(timeout))

	if err := tlsConn.Handshake(); err != nil {
		return nil, fmt.Errorf("TLS handshake: %w", err)
	}

	certs := tlsConn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return nil, fmt.Errorf("no certificates received after STARTTLS")
	}
	return certs, nil
}
