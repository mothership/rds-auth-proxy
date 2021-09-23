package pg

import (
	"crypto/tls"
	"crypto/x509"
	"net"

	"github.com/jackc/pgproto3/v2"
)

// SSLMode is the type of SSL required
// https://www.postgresql.org/docs/8.4/libpq-connect.html#LIBPQ-CONNECT-SSLMODE
type SSLMode string

const (
	// SSLDisabled only tries a non-SSL connection
	SSLDisabled SSLMode = "disable"
	// SSLAllow first try a non-SSL connection, if that fails, tries an SSL connection
	// XXX: Not allowed at this time
	SSLAllow = "allow"
	// SSLPreferred is like allow, but tries an SSL connection first -- default behavior of psql
	SSLPreferred = "preferred"
	// SSLRequired only tries an SSL connection. If a root CA file is present, verify the certificate in the same way as if verify-ca was specified
	SSLRequired = "require"
	// SSLVerifyCA only tries an SSL connection, and verifies that the server certificate is issued by a trusted CA.
	SSLVerifyCA = "verify-ca"
	// SSLVerifyFull only tries an SSL connection, verifies that the server certificate is issued by a trusted CA and that
	// the server hostname matches that in the certificate.
	SSLVerifyFull = "verify-full"
)

// Connect connects to an upstream database
func Connect(host string, mode SSLMode, cert *tls.Certificate, rootCert *x509.Certificate) (net.Conn, error) {
	connection, err := net.Dial("tcp", host)
	if err != nil {
		return nil, err
	}

	backend := pgproto3.NewFrontend(pgproto3.NewChunkReader(connection), connection)
	if mode != SSLDisabled {
		// log.Info("SSL connections are enabled.")

		/*
		 * First determine if SSL is allowed by the backend. To do this, send an
		 * SSL request. The response from the backend will be a single byte
		 * message. If the value is 'S', then SSL connections are allowed and an
		 * upgrade to the connection should be attempted. If the value is 'N',
		 * then the backend does not support SSL connections.
		 */
		sslRequest := &pgproto3.SSLRequest{}
		err = backend.Send(sslRequest)
		if err != nil {
			return nil, err
		}

		response := make([]byte, 4096)
		_, err = connection.Read(response)
		if err != nil {
			return nil, err
		}

		if len(response) > 0 && response[0] == SSLAllowed {
			// TODO: should probably decide whether or not to error based on SSL mode
			//       but we'll pass the error back anyhow
			connection, err = UpgradeClient(host, connection, mode, cert, rootCert)
		} else if mode != SSLPreferred {
			// Close the connection only if we wanted required or higher
			connection.Close()
		}
	}

	return connection, err
}

// UpgradeServer upgrades a server connection with SSL
func UpgradeServer(client net.Conn, cert *tls.Certificate) net.Conn {
	if cert == nil {
		return client
	}
	tlsConfig := tls.Config{}
	tlsConfig.Certificates = []tls.Certificate{*cert}
	return tls.Server(client, &tlsConfig)
}

// UpgradeClient upgrades a client connection with SSL
func UpgradeClient(hostPort string, connection net.Conn, mode SSLMode, cert *tls.Certificate, rootCert *x509.Certificate) (net.Conn, error) {
	if mode == SSLDisabled {
		return connection, nil
	}

	tlsConfig := tls.Config{}
	if mode == SSLPreferred || mode == SSLRequired || mode == SSLVerifyCA {
		tlsConfig.InsecureSkipVerify = true
	}

	if mode == SSLVerifyFull {
		hostname, _, err := net.SplitHostPort(hostPort)
		if err != nil {
			return connection, err
		}
		tlsConfig.ServerName = hostname
	}

	var err error
	tlsConfig.Certificates = []tls.Certificate{*cert}
	tlsConfig.RootCAs, err = x509.SystemCertPool()
	if err != nil {
		return connection, err
	}

	if rootCert != nil {
		tlsConfig.RootCAs.AddCert(rootCert)
	}

	// do the upgrade
	client := tls.Client(connection, &tlsConfig)
	if mode == SSLVerifyCA || (mode == SSLRequired && rootCert != nil) {
		err := verifyCA(client, &tlsConfig)
		if err != nil {
			return connection, err
		}
	}

	return client, nil
}

// verifyCA explicitly does the handshake and certificate chain validation in the case that we need to validate
// the CA, or we have a CA cert to validate against and the mode is require.
func verifyCA(client *tls.Conn, tlsConf *tls.Config) error {
	err := client.Handshake()
	if err != nil {
		return err
	}

	// Get the peer/CA certificates from the connection state.
	peerCerts := client.ConnectionState().PeerCertificates
	caCert := peerCerts[0]
	peerCerts = peerCerts[1:]

	options := x509.VerifyOptions{
		DNSName:       client.ConnectionState().ServerName,
		Intermediates: x509.NewCertPool(),
		Roots:         tlsConf.RootCAs,
	}
	// build the intermediate chain for verification
	for _, certificate := range peerCerts {
		options.Intermediates.AddCert(certificate)
	}

	// verify the CA cert is legitimate by building a path between it and the root
	// certificates we have, using the intermediates provided by the peer certificates.
	_, err = caCert.Verify(options)
	return err
}
