package cert

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
)

type Info struct {
	PinSHA256  string
	ServerName string
}

func Read(path string) (Info, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Info{}, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return Info{}, fmt.Errorf("invalid certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return Info{}, err
	}

	sum := sha256.Sum256(cert.Raw)

	info := Info{PinSHA256: hex.EncodeToString(sum[:]), ServerName: cert.Subject.CommonName}
	if len(cert.DNSNames) > 0 {
		info.ServerName = cert.DNSNames[0]
	}

	return info, nil
}

func CertificatePin(path string) (string, error) {
	info, err := Read(path)
	return info.PinSHA256, err
}
