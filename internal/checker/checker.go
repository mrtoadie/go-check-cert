package checker

import (
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"

)
type CertInfo struct {
	URL           string
	Issuer        string
	NotBefore     time.Time
	NotAfter      time.Time
	DaysRemaining int
	Status        string
	Error         error
}

// connects to the host and extracts certificate data
func CheckCertificate(url string, timeout time.Duration) CertInfo {
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
	info.Issuer = cert.Issuer.CommonName
	info.NotBefore, info.NotAfter = cert.NotBefore, cert.NotAfter
	info.DaysRemaining = int(info.NotAfter.Sub(time.Now()).Hours() / 24)

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
