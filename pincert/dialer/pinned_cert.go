package dialer

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"golang.org/x/xerrors"
	"net"
)

// TLSDialer is a function for creating TLS connections for non-proxied
// requests that can be assigned to a http.Transports's DialTLS field.
type TLSDialer func(network, addr string) (net.Conn, error)

// WithPinnedCertVerification returns a TLS dialer function which checks that the remote
// server provides a certificate whose SHA256 fingerprint matches the provided value
func WithPinnedCertVerification(pkFingerprint []byte, tlsConfig *tls.Config) TLSDialer {
	return func(network, addr string) (net.Conn, error) {
		// Establish a TLS connection to the remote server and verify
		// all presented TLS certificate
		conn, err := tls.Dial(network, addr, tlsConfig)
		if err != nil {
			return nil, err
		}
		if err := verifyPinnedCert(pkFingerprint, conn.ConnectionState().PeerCertificates); err != nil {
			_ = conn.Close()
			return nil, err
		}
		return conn, err
	}

}

func verifyPinnedCert(pkFingerprint []byte, certificates []*x509.Certificate) error {
	for _, cert := range certificates {
		certDER, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
		if err != nil {
			return xerrors.Errorf("unable to serialize certificate public key: %w", err)
		}
		fingerprint := sha256.Sum256(certDER)
		//Matched cert PK pkFingerprint to the one provided
		if bytes.Equal(fingerprint[:], pkFingerprint) {
			return nil
		}
	}
	return xerrors.Errorf("remote server presented a certificate which does not match the provided fingerprint")
}
