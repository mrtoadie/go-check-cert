// internal/checker/checker.go
package checker

import (
	//"crypto/x509"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"time"

	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
)

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
	KeyAlgorithm       string   // e.g. RSA, ECDSA
	KeySize            int      // e.g. 2048, 256
	SignatureAlgorithm string   // e.g. SHA256-RSA
	SANs               []string //list of alternate names
	ThumbprintSHA1     string
	ThumbprintSHA256   string
}

// connects to the host and extracts certificate data
func CheckCertExpiry(url string, timeout time.Duration) CertInfo {
	info := CertInfo{URL: url}

	// URL bereinigen
	url = strings.TrimPrefix(strings.TrimPrefix(url, "https://"), "http://")
	if !strings.Contains(url, ":") {
		url += ":443"
	}

	// TLS handshake
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: timeout}, "tcp", url, &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		info.Error, info.Status = err, "ERROR"
		return info
	}
	defer conn.Close()

	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		info.Error, info.Status = fmt.Errorf("no certificates found"), "ERROR"
		return info
	}

	cert := certs[0]
	//info.Issuer = cert.Issuer.CommonName
	info.Issuer = cert.Issuer.String()
	// new
	info.Subject = cert.Subject.CommonName
	info.SerialNumber = cert.SerialNumber.String()
	//
	info.NotBefore, info.NotAfter = cert.NotBefore, cert.NotAfter
	info.DaysRemaining = int(info.NotAfter.UTC().Sub(time.Now().UTC()).Hours() / 24)

	// key info
	switch pub := cert.PublicKey.(type) {
	case *rsa.PublicKey:
		info.KeyAlgorithm = "RSA"
		info.KeySize = pub.Size() * 8 // Bits
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

	// signature algorithm
	info.SignatureAlgorithm = cert.SignatureAlgorithm.String()

	// SANs (subject alternative names)
	for _, dnsName := range cert.DNSNames {
		info.SANs = append(info.SANs, dnsName)
	}
	for _, ip := range cert.IPAddresses {
		info.SANs = append(info.SANs, ip.String())
	}
	for _, uri := range cert.URIs {
		info.SANs = append(info.SANs, uri.String())
	}

	// thumbprints hashes
	hashSHA1 := sha1.Sum(cert.Raw)
	hashSHA256 := sha256.Sum256(cert.Raw)

	info.ThumbprintSHA1 = hex.EncodeToString(hashSHA1[:])
	info.ThumbprintSHA256 = hex.EncodeToString(hashSHA256[:])
	//

	// determine status
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
