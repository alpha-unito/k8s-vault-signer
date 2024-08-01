package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alpha-unito/k8s-vault-signer/internal/controller/certificates/signer"
	"github.com/alpha-unito/k8s-vault-signer/pkg/config"
	vault "github.com/alpha-unito/k8s-vault-signer/pkg/vault/client"
	"github.com/alpha-unito/k8s-vault-signer/pkg/vault/sign"
	"github.com/alpha-unito/k8s-vault-signer/pkg/version"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/component-base/cli"
	"k8s.io/klog/v2"
)

var c = config.NewConfig()

func main() {
	cmd := &cobra.Command{
		Use:   "vault-signer",
		Short: "Vault CSR signer for Kubernetes",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithCancel(context.Background())

			vclient, err := vault.NewClient(c.VaultAddress)
			if err != nil {
				klog.Exitf("error creating Vault client: %s", err)
			}

			authenticator, err := vault.NewAuthenticator(c.VaultAuthConfig)
			if err != nil {
				klog.Exitf("error creating Vault authenticator: %s", err)
			}

			secret, err := authenticator.Authenticate(ctx, vclient)
			if err != nil {
				klog.Exitf("error authenticating with Vault: %s", err)
			}

			watcher, err := vault.NewWatcher(authenticator, vclient, secret)
			if err != nil {
				klog.Exitf("error creating token watcher: %s", err)
			}

			cfg, err := clientcmd.BuildConfigFromFlags("", c.Kubeconfig)
			if err != nil {
				klog.Exitf("error building kubernetes config from flags: %s", err)
			}

			kclient, err := kubernetes.NewForConfig(cfg)
			if err != nil {
				klog.Exitf("error creating kubernetes client from config: %s", err)
			}

			factory := informers.NewSharedInformerFactory(kclient, 5*time.Minute)
			csrInformer := factory.Certificates().V1().CertificateSigningRequests()

			vaultSigner, err := sign.NewSigner(vclient, c.VaultPki, c.VaultRole)
			if err != nil {
				klog.Exitf("error creating Vault signer: %s", err)
			}

			controller, err := signer.NewVaultCSRSigningController(
				ctx,
				kclient,
				csrInformer,
				vaultSigner,
				c.SigningDuration.Duration,
			)
			if err != nil {
				klog.Fatalf("error creating auth signing controller: %s", err)
			}

			go csrInformer.Informer().Run(ctx.Done())
			go controller.Run(ctx, 5)
			go watcher.Watch(ctx, vclient)

			sigterm := make(chan os.Signal)
			signal.Notify(sigterm, os.Interrupt, syscall.SIGTERM)

			select {
			case <-sigterm:
				klog.Info("received SIGTERM, terminating gracefully")
			case <-ctx.Done():
				klog.Info("certificate controller terminated correctly")
			}

			cancel()
		},
		Version: version.Version,
	}

	c.AddFlags(pflag.CommandLine)
	if err := c.Validate(); err != nil {
		klog.Exitf("error validating configuration: %s", err)
	}

	code := cli.Run(cmd)
	os.Exit(code)
}
