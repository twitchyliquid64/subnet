package cert

import (
	"bytes"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"time"
)

type blacklistEntry struct {
	Justification string `json:"justification"`
	PublicKey     []byte `json:"public_key"`
	AddedEpoch    int64  `json:"timestamp"`
}

var crl []blacklistEntry

// readCRL ingests the CRL at path.
func readCRL(path string) ([]blacklistEntry, error) {
	var temp []blacklistEntry
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	d := json.NewDecoder(f)
	if err := d.Decode(&temp); err != nil {
		return nil, err
	}
	return temp, nil
}

// AddToCRL inserts an entry into the CRL at crlPath. The private key is read from the
// PEM-encoded cert at certPath.
func AddToCRL(crlPath, certPath, justification string) error {
	pemBytes, err := ioutil.ReadFile(certPath)
	if err != nil {
		return err
	}
	certDERBlock, _ := pem.Decode(pemBytes)
	if certDERBlock == nil {
		return errors.New("No certificate data read from PEM")
	}
	cert, err := x509.ParseCertificate(certDERBlock.Bytes)
	if err != nil {
		return err
	}
	pubKey, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
	if err != nil {
		return err
	}

	tempCrl, err := readCRL(crlPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	tempCrl = append(tempCrl, blacklistEntry{
		Justification: justification,
		PublicKey:     pubKey,
		AddedEpoch:    time.Now().Unix(),
	})

	crlData, err := json.MarshalIndent(tempCrl, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(crlPath, crlData, 0755)
}

// InitCRL reads the CRL from disk and starts the routine to periodically refresh it.
func InitCRL(path string) error {
	c, err := readCRL(path)
	if err != nil {
		return err
	}
	crl = c
	go func() {
		for {
			time.Sleep(time.Minute * 2)
			c, err := readCRL(path)
			if err == nil {
				crl = c
			} else {
				fmt.Fprintf(os.Stderr, "Failed to read CRL file: %s", err)
			}
		}
	}()
	return nil
}

// CheckCRL returns an error if cert is on the CRL.
func CheckCRL(cert *x509.Certificate) error {
	if len(crl) == 0 {
		return nil
	}
	pubKey, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
	if err != nil {
		return err
	}

	for i, c := range crl {
		if bytes.Compare(c.PublicKey, pubKey) == 0 {
			return fmt.Errorf("CRL match at index %d - Justification %q", i, c.Justification)
		}
	}
	return nil
}
