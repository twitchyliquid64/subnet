package conn

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
)

// TLSConfig generates and returns a TLS configuration based on the given parameters.
// If certPemPath is empty, no Certificate is set on the config.
// If caCertPath is empty, no trust root is established and no client/serv verification
// is performed.
func TLSConfig(certPemPath, keyPemPath, caCertPath string) (*tls.Config, error) {
	roots := x509.NewCertPool()
	if caCertPath != "" {
		pemBytes, err := ioutil.ReadFile(caCertPath)
		if err != nil {
			return nil, err
		}

		roots.AppendCertsFromPEM(pemBytes)
	}

	gTLSConfig := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		RootCAs:                  roots,
		ClientCAs:                roots,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}

	if certPemPath != "" {
		mainCert, err := tls.LoadX509KeyPair(certPemPath, keyPemPath)
		if err != nil {
			return nil, err
		}
		gTLSConfig.Certificates = []tls.Certificate{mainCert}
	}

	if caCertPath == "" {
		gTLSConfig.InsecureSkipVerify = true
		log.Println("Warning: No CA certificate specified. Skipping TLS verification of server. This is bad!")
	} else {
		gTLSConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}

	return gTLSConfig, nil
}
