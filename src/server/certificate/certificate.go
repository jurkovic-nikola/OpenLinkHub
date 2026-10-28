package certificate

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// EnsureCertificate will create a self-signed certificate if no certificate exists
func EnsureCertificate(directory string, hosts []string) (tls.Certificate, error) {
	if err := os.MkdirAll(directory, 0700); err != nil {
		return tls.Certificate{}, err
	}

	certPath := filepath.Join(directory, "server.crt")
	keyPath := filepath.Join(directory, "server.key")

	_, certErr := os.Stat(certPath)
	_, keyErr := os.Stat(keyPath)

	if certErr == nil || keyErr == nil {
		return tls.LoadX509KeyPair(certPath, keyPath)
	}

	if !os.IsNotExist(certErr) {
		return tls.Certificate{}, certErr
	}
	if !os.IsNotExist(keyErr) {
		return tls.Certificate{}, keyErr
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return tls.Certificate{}, err
	}

	now := time.Now()
	certificate := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: "OpenLinkHub",
		},
		NotBefore:             now.Add(-5 * time.Minute),
		NotAfter:              now.AddDate(2, 0, 0),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	for _, host := range hosts {
		if host == "" {
			continue
		}

		if ip := net.ParseIP(host); ip != nil {
			certificate.IPAddresses = append(certificate.IPAddresses, ip)
		} else {
			certificate.DNSNames = append(certificate.DNSNames, host)
		}
	}

	if len(certificate.IPAddresses) == 0 && len(certificate.DNSNames) == 0 {
		return tls.Certificate{}, fmt.Errorf("no certificate hostnames or IP addresses provided")
	}

	der, err := x509.CreateCertificate(
		rand.Reader,
		&certificate,
		&certificate,
		&privateKey.PublicKey,
		privateKey,
	)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: der,
	})
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	if err := writeCertificateFile(keyPath, keyPEM); err != nil {
		return tls.Certificate{}, err
	}
	if err := writeCertificateFile(certPath, certPEM); err != nil {
		return tls.Certificate{}, err
	}

	return tls.X509KeyPair(certPEM, keyPEM)
}

// writeCertificateFile will write a new certificate file without replacing an existing file
func writeCertificateFile(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}

	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}

	return file.Close()
}
