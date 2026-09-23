package cert

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
)

func CertificatePin(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return "", fmt.Errorf("invalid certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256(cert.Raw)

	return hex.EncodeToString(sum[:]), nil
}
