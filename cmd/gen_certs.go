package cmd

import (
	"fmt"

	"github.com/mothership/rds-auth-proxy/pkg/cert"
	"github.com/mothership/rds-auth-proxy/pkg/file"
	"github.com/spf13/cobra"
)

var genCertsCommand = &cobra.Command{
	Use:   "gen-cert",
	Short: "Generates a self-signed certificate",
	Long:  `Generates a self-signed certificate`,
	RunE: func(cmd *cobra.Command, args []string) error {
		keyPath, err := cmd.Flags().GetString("key")
		if err != nil {
			return err
		}
		if keyPath == "" {
			return fmt.Errorf("Key path must not be empty")
		}

		certPath, err := cmd.Flags().GetString("certificate")
		if err != nil {
			return err
		}
		if certPath == "" {
			return fmt.Errorf("Certificate path must not be empty")
		}

		if file.Exists(certPath) || file.Exists(keyPath) {
			return fmt.Errorf("certificate/key already exists at this location")
		}

		hosts, err := cmd.Flags().GetString("hosts")
		if err != nil {
			return err
		}

		certBytes, keyBytes, err := cert.GenerateSelfSignedCert(hosts, false)
		if err != nil {
			return err
		}
		err = cert.Save(certPath, certBytes)
		if err != nil {
			return err
		}
		return cert.Save(keyPath, keyBytes)
	},
}

func init() {
	rootCmd.AddCommand(genCertsCommand)
	genCertsCommand.PersistentFlags().String("certificate", "", "Path to generate the certificate")
	_ = genCertsCommand.MarkPersistentFlagRequired("certificate")
	genCertsCommand.PersistentFlags().String("key", "", "Path to generate the private key")
	_ = genCertsCommand.MarkPersistentFlagRequired("key")

	genCertsCommand.PersistentFlags().String("hosts", "rds-auth-proxy", "Comma separated list of hosts to add to the certificate")
}
