// internal/checker/checker.go
// last modification: Apr 28 2026
package checker

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

// CertInfo holds the extracted certificate details and status
type CertInfo struct {
	URL                string
	Issuer             string
	Subject            string
	SerialNumber       string
	NotBefore          time.Time
	NotAfter           time.Time
	DaysRemaining      int
	Status             string
	Error              error
	KeyAlgorithm       string
	KeySize            int
	SignatureAlgorithm string
	SANs               []string
	ChainLength        int
	IsChainComplete    bool
	ChainError         string
	IsSelfSigned       bool
	RootIssuer         string
}

// CheckCertExpiry is the public entry point
// delegates to specialized functions based on whether the target is a local file or a remote URL
func CheckCertExpiry(target string, hostname string, timeout time.Duration) CertInfo {
	// Decide: File or Remote?
	if IsFilePath(target) {
		return checkLocalFile(target)
	}
	return checkRemoteCert(target, hostname, timeout)
}

// checkRemoteCert handles TLS connections to remote hosts.
func checkRemoteCert(target string, hostname string, timeout time.Duration) CertInfo {
	info := CertInfo{URL: target}

	// clean URL for connection
	url := strings.TrimPrefix(strings.TrimPrefix(target, "https://"), "http://")
	if !strings.Contains(url, ":") {
		url += ":443"
	}

	// extract hostname if not provided
	if hostname == "" {
		hostname = ExtractHostname(target)
	}

	if hostname == "" {
		info.Error = fmt.Errorf("failed to extract hostname")
		info.Status = "ERROR"
		return info
	}

	// establish TLS connection
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: timeout}, "tcp", url, &tls.Config{
		InsecureSkipVerify: false, // better false (https://cwe.mitre.org/data/definitions/295.html)
		ServerName:         hostname,
	})
	if err != nil {
		info.Error = err
		info.Status = "ERROR"
		return info
	}
	defer conn.Close()

	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		info.Error = fmt.Errorf("no certificates found")
		info.Status = "ERROR"
		return info
	}

	// extract info (pass the full chain for validation)
	return extractCertInfo(certs[0], target, certs, hostname)
}

// checkLocalFile handles reading and parsing local certificate files
func checkLocalFile(filePath string) CertInfo {
	info := CertInfo{URL: filePath}

	data, err := os.ReadFile(filePath)
	if err != nil {
		info.Error = err
		info.Status = "ERROR"
		return info
	}

	// decode PEM
	block, _ := pem.Decode(data)
	if block == nil {
		info.Error = fmt.Errorf("invalid PEM format")
		info.Status = "ERROR"
		return info
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		info.Error = err
		info.Status = "ERROR"
		return info
	}

	// for local files, chain is nil (single cert)
	return extractCertInfo(cert, filePath, nil, "")
}

// extractCertInfo contains the shared logic for extracting metadata from a certificate
func extractCertInfo(cert *x509.Certificate, source string, chain []*x509.Certificate, hostname string) CertInfo {
	info := CertInfo{
		URL:           source,
		Issuer:        cert.Issuer.CommonName,
		Subject:       cert.Subject.CommonName,
		SerialNumber:  cert.SerialNumber.String(),
		NotBefore:     cert.NotBefore,
		NotAfter:      cert.NotAfter,
		DaysRemaining: int(cert.NotAfter.UTC().Sub(time.Now().UTC()).Hours() / 24),
	}

	// chain logic
	if chain != nil && len(chain) > 0 {
		info.ChainLength = len(chain)
		info.IsChainComplete = true
		// root issuer is the last cert in the chain
		info.RootIssuer = chain[len(chain)-1].Issuer.CommonName
	} else {
		// local file or single cert
		info.ChainLength = 1
		info.IsChainComplete = true
		info.IsSelfSigned = cert.Issuer.String() == cert.Subject.String()
		info.RootIssuer = cert.Issuer.CommonName
	}

	// key info
	switch pub := cert.PublicKey.(type) {
	case *rsa.PublicKey:
		info.KeyAlgorithm = "RSA"
		info.KeySize = pub.Size() * 8
	case *ecdsa.PublicKey:
		info.KeyAlgorithm = "ECDSA"
		info.KeySize = pub.Curve.Params().BitSize
	case *ed25519.PublicKey:
		info.KeyAlgorithm = "Ed25519"
		info.KeySize = 256
	default:
		info.KeyAlgorithm = "Unknown"
		info.KeySize = 0
	}

	info.SignatureAlgorithm = cert.SignatureAlgorithm.String()

	// SANs
	for _, dnsName := range cert.DNSNames {
		info.SANs = append(info.SANs, dnsName)
	}
	for _, ip := range cert.IPAddresses {
		info.SANs = append(info.SANs, ip.String())
	}

	// status determination
	if info.DaysRemaining < 0 {
		info.Status = "EXPIRED"
	} else if info.DaysRemaining < 30 {
		info.Status = "WARNING"
	} else if info.DaysRemaining < 60 {
		info.Status = "SOON"
	} else {
		info.Status = "OK"
	}

	return info
}
