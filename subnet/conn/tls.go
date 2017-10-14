package conn

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"log"
	"github.com/twitchyliquid64/subnet/subnet/cert"
	"time"
)

// TLSConfig generates and returns a TLS configuration based on the given parameters.
// If certPemPath is empty, no Certificate is set on the config.
// If caCertPath is empty, no trust root is established and no client/serv verification
// is performed.
func TLSConfig(certPemPath, keyPemPath, caCertPath string) (*tls.Config, error) {
	var caCertParsed *x509.Certificate
	if caCertPath != "" {
		pemBytes, err := ioutil.ReadFile(caCertPath)
		if err != nil {
			return nil, err
		}
		certDERBlock, _ := pem.Decode(pemBytes)
		if certDERBlock == nil {
			return nil, errors.New("No certificate data read from PEM")
		}
		caCertParsed, err = x509.ParseCertificate(certDERBlock.Bytes)
		if err != nil {
			return nil, err
		}
	}

	gTLSConfig := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			if caCertParsed == nil {
				return nil //perform no verification
			}
			for _, c := range rawCerts {
				parsedCert, err := x509.ParseCertificate(c)
				if err != nil {
					return err
				}
				certErr := parsedCert.CheckSignatureFrom(caCertParsed)
				if parsedCert.NotAfter.Before(time.Now()) || parsedCert.NotBefore.After(time.Now()) {
					certErr = errors.New("Certificate expired or used too soon")
				}
				if certErr == nil {
					if crlErr := cert.CheckCRL(parsedCert); crlErr != nil {
						certErr = crlErr
					}
				}
				log.Printf("Remote presented certificate %d with time bounds (%v-%v). Verification error for certificate: %+v", parsedCert.SerialNumber, parsedCert.NotBefore, parsedCert.NotAfter, certErr)
				return certErr
			}
			return errors.New("Expected certificate which would pass, none presented")
		},
		InsecureSkipVerify: true,
	}

	if certPemPath != "" {
		mainCert, err := tls.LoadX509KeyPair(certPemPath, keyPemPath)
		if err != nil {
			return nil, err
		}
		gTLSConfig.Certificates = []tls.Certificate{mainCert}
	}

	if caCertPath == "" {
		log.Println("Warning: No CA certificate specified. Skipping TLS verification of server. This is bad!")
	} else {
		gTLSConfig.ClientAuth = tls.RequestClientCert
	}

	return gTLSConfig, nil
}
