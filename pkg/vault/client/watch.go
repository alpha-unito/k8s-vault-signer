package client

import (
	"context"
	"fmt"

	vault "github.com/hashicorp/vault/api"

	"k8s.io/klog/v2"
)

type Watcher struct {
	authenticator *Authenticator
	watcher       *vault.LifetimeWatcher
}

func NewWatcher(a *Authenticator, vclient *vault.Client, secret *vault.Secret) (*Watcher, error) {
	watcher, err := lifetimeWatcher(vclient, secret)
	if err != nil {
		return nil, fmt.Errorf("failed to create Vault lifetime watcher: %s", err)
	}

	return &Watcher{
		authenticator: a,
		watcher:       watcher,
	}, nil
}

func (w *Watcher) Watch(ctx context.Context, vclient *vault.Client) {
	for {
		w.watch(ctx)

		logger := klog.FromContext(ctx)
		logger.V(4).Info("retry logging into Vault")

		secret, err := w.authenticator.Authenticate(ctx, vclient)
		if err != nil {
			klog.Exitf("error authenticating with Vault: %s", err)
		}

		watcher, err := lifetimeWatcher(vclient, secret)
		if err != nil {
			klog.Exitf("failed to create Vault lifetime watcher: %s", err)
		}

		w.watcher = watcher
	}
}

func (w *Watcher) watch(ctx context.Context) {
	go w.watcher.Start()
	defer w.watcher.Stop()

	logger := klog.FromContext(ctx)
	for {
		select {
		case err := <-w.watcher.DoneCh():
			if err != nil {
				logger.V(4).Error(err, "failed to renew Vault token")
			} else {
				logger.V(4).Info("token can no longer be renewed")
			}
			return

		case _ = <-w.watcher.RenewCh():
			logger.V(4).Info("succesfully renewed Vault token")
		}
	}
}

func lifetimeWatcher(vclient *vault.Client, secret *vault.Secret) (*vault.LifetimeWatcher, error) {
	if ok, err := secret.TokenIsRenewable(); !ok {
		if err != nil {
			return nil, fmt.Errorf("failed to check Vault token: %s", err)
		}
		return nil, fmt.Errorf("secret is not renewable")
	}

	watcher, err := vclient.NewLifetimeWatcher(&vault.LifetimeWatcherInput{
		Secret:    secret,
		Increment: secret.Auth.LeaseDuration,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to initialize new lifetime watcher for renewing auth token: %w", err)
	}

	return watcher, nil
}
