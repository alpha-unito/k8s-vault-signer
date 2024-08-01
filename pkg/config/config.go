package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/pflag"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

type Config struct {
	Kubeconfig      string
	SigningDuration metav1.Duration
	VaultAddress    string
	VaultAuthConfig string
	VaultPki        string
	VaultRole       string
}

func NewConfig() *Config {
	envDuration := os.Getenv("SIGNING_DURATION")
	signingDuration, err := time.ParseDuration(envDuration)
	if err != nil {
		if envDuration != "" {
			klog.Infof("failed to parse SIGNING_DURATION: %v", err)
		}
		signingDuration = 365 * 24 * time.Hour
	}

	return &Config{
		Kubeconfig:      os.Getenv("KUBECONFIG"),
		SigningDuration: metav1.Duration{Duration: signingDuration},
		VaultAddress:    os.Getenv("VAULT_ADDR"),
		VaultAuthConfig: os.Getenv("VAULT_AUTH_CONFIG"),
		VaultPki:        os.Getenv("VAULT_PKI"),
		VaultRole:       os.Getenv("VAULT_ROLE"),
	}
}

func (c *Config) Validate() error {
	var errorsFound bool

	if c.VaultAddress == "" {
		errorsFound = true
		klog.Errorf("please specify --vault-address or set the VAULT_ADDR environment variable")
	}
	if c.VaultAuthConfig == "" {
		errorsFound = true
		klog.Errorf("please specify --vault-auth-config or set the VAULT_AUTH_CONFIG environment variable")
	}
	if c.VaultPki == "" {
		errorsFound = true
		klog.Errorf("please specify --vault-pki or set the VAULT_PKI environment variable")
	}
	if c.VaultRole == "" {
		errorsFound = true
		klog.Errorf("please specify --vault-role or set the VAULT_ROLE environment variable")
	}

	if errorsFound {
		return fmt.Errorf("failed to validate input parameters")
	}
	return nil
}

func (c *Config) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.Kubeconfig, "kubeconfig", c.Kubeconfig, "Absolute path to the kubeconfig file. If the service is running inside a Pod, this option is not necessary: the in-cluster config will be used by default.")
	fs.DurationVar(&c.SigningDuration.Duration, "signing-duration", c.SigningDuration.Duration, "The length of duration signed certificates will be given.")
	fs.StringVar(&c.VaultAddress, "vault-address", c.VaultAddress, "Address of the Vault cluster.")
	fs.StringVar(&c.VaultAuthConfig, "vault-auth-config", c.VaultAuthConfig, "Path of the Vault authentication configuration file.")
	fs.StringVar(&c.VaultPki, "vault-pki", c.VaultPki, "Path of the Vault PKI secret mount used to generate the CA.")
	fs.StringVar(&c.VaultRole, "vault-role", c.VaultRole, "Name of the Vault role used to sign the certificates.")
}
