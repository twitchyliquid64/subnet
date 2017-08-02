package cert

import (
  "crypto/rand"
  "crypto/rsa"
  "crypto/x509"
  "crypto/x509/pkix"
  "encoding/binary"
  "encoding/pem"
  "errors"
  "io/ioutil"
  "math/big"
  unsecure_rand "math/rand"
  "os"
  "time"
)

// ErrInsecureKeyBitSize is returned if a generate method is called with too few bits.
var ErrInsecureKeyBitSize = errors.New("too few bits when generating key")

// LoadPrivateCertPEM returns a certificate and private key, decoded from bytesCert (PEM) and keyBytes (PEM).
func LoadPrivateCertPEM(bytesCert []byte, keyBytes []byte) (*x509.Certificate, *rsa.PrivateKey, error) {
	certDERBlock, _ := pem.Decode(bytesCert)
	if certDERBlock == nil {
		return nil, nil, errors.New("No certificate data read from PEM")
	}
	cert, err := x509.ParseCertificate(certDERBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}

	keyBlock, _ := pem.Decode(keyBytes)
	if keyBlock == nil {
		return nil, nil, errors.New("No key data read from PEM")
	}
	priv, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}
	return cert, priv, nil
}

// LoadPrivateCertFromFilePEM returns a cert & PK after loading both those components from the files at the specified paths.
// certPath should point to a PEM encoded certificate, and keyPath should point to a PEM encoded private key.
func LoadPrivateCertFromFilePEM(certPath, keyPath string) (*x509.Certificate, *rsa.PrivateKey, error) {
	certBytes, err := ioutil.ReadFile(certPath)
	if err != nil {
		return nil, nil, err
	}
	keyBytes, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return nil, nil, err
	}
	return LoadPrivateCertPEM(certBytes, keyBytes)
}

// GenerateRSA returns a RSA private key with the given key length.
func GenerateRSA(bitSize int) (*rsa.PrivateKey, error) {
	if bitSize <= 1024 {
		return nil, ErrInsecureKeyBitSize
	}

	return rsa.GenerateKey(rand.Reader, bitSize)
}


func makeBasicCert(now time.Time) *x509.Certificate {
	//Make a subjectKeyId. There are no security requirements for this field, but the
	//more statistically distributed it is the better it can be used.
	subjectKeyNum := uint64(unsecure_rand.Int63())
	var subjectKeyBytes = make([]byte, 16)
	binary.PutUvarint(subjectKeyBytes, subjectKeyNum)

	return &x509.Certificate{
		SerialNumber: big.NewInt(int64(unsecure_rand.Int63())),
		Subject: pkix.Name{
			Country:            []string{"U.S"},
			Organization:       []string{"Acme Co."},
			OrganizationalUnit: []string{"Acme Co." + "U"},
		},
		NotBefore:    now,
		NotAfter:     now.AddDate(0, 6, 0), //6 month expiry
		SubjectKeyId: subjectKeyBytes[:5],
	}
}

func IssueClientCert(CACertPath, CAKeyPath, clientCertPath, clientKeyPath string) error {
  unsecure_rand.Seed(time.Now().Unix())
  now := time.Now()

  // -- make the cert --
  //Make a subjectKeyId. There are no security requirements for this field, but the
	//more statistically distributed it is the better it can be used.
	cert := makeBasicCert(now)
  cert.IsCA = false
  cert.BasicConstraintsValid = true
  cert.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth}
  cert.KeyUsage = x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature

  // -- make the key --
  key, err := GenerateRSA(2048)
	if err != nil {
		return err
	}

  // -- read CA cert --
  ca, caKey, err := LoadPrivateCertFromFilePEM(CACertPath, CAKeyPath)
  if err != nil {
    return err
  }

  // -- do signature --
  cert.Issuer = ca.Subject
  fullCertBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, &key.PublicKey, caKey)
  if err != nil {
    return err
  }

  // -- write the cert/key to disk --
  certFile, err := os.Create(clientCertPath)
  if err != nil {
    return err
  }
  pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: fullCertBytes})
  certFile.Close()
  keyFile, err := os.Create(clientKeyPath)
  if err != nil {
    return err
  }
  pem.Encode(keyFile, &pem.Block{Type: "RSA PRIVATE KEY",
    Bytes: x509.MarshalPKCS1PrivateKey(key)})
  keyFile.Close()
  return nil
}

// MakeServerCert generates a CA+Server certificate and writes it into the specified paths.
func MakeServerCert(serverCertPath, serverKeyPath, CACertPath, CAKeyPath string) error {
  unsecure_rand.Seed(time.Now().Unix())
  now := time.Now()

  // -- make the cert --
  //Make a subjectKeyId. There are no security requirements for this field, but the
	//more statistically distributed it is the better it can be used.
	cert := makeBasicCert(now)
  cert.IsCA = false
  cert.BasicConstraintsValid = true
  cert.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth}
  cert.KeyUsage = x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature

  // -- make the key --
  key, err := GenerateRSA(2048)
	if err != nil {
		return err
	}
  // -- end server cert/key generation --

  // -- make the CA --
  ca := makeBasicCert(now)
  ca.IsCA = true
  ca.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth}
	ca.KeyUsage |= x509.KeyUsageCertSign

  // -- make the CA key --
  caKey, err := GenerateRSA(2048)
  if err != nil {
    return err
  }
  fullCABytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caKey.PublicKey, caKey)
  if err != nil {
    return err
  }

  // -- finally, sign the server cert with the CA cert --
  cert.Issuer = ca.Subject
  fullCertBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, &key.PublicKey, caKey)
  if err != nil {
    return err
  }

  // -- write the server cert/key to disk --
  certFile, err := os.Create(serverCertPath)
  if err != nil {
    return err
  }
  pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: fullCertBytes})
  certFile.Close()
  keyFile, err := os.Create(serverKeyPath)
  if err != nil {
    return err
  }
  pem.Encode(keyFile, &pem.Block{Type: "RSA PRIVATE KEY",
    Bytes: x509.MarshalPKCS1PrivateKey(key)})
  keyFile.Close()

  // -- write the CA cert/key to disk --
  certCAFile, err := os.Create(CACertPath)
  if err != nil {
    return err
  }
  pem.Encode(certCAFile, &pem.Block{Type: "CERTIFICATE", Bytes: fullCABytes})
  certFile.Close()
  keyCAFile, err := os.Create(CAKeyPath)
  if err != nil {
    return err
  }
  pem.Encode(keyCAFile, &pem.Block{Type: "RSA PRIVATE KEY",
    Bytes: x509.MarshalPKCS1PrivateKey(caKey)})
  keyFile.Close()

  return nil
}
