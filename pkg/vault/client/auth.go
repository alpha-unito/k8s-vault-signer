package client

import (
	"context"
	"fmt"

	vault "github.com/hashicorp/vault/api"
	auth "github.com/hashicorp/vault/api/auth/approle"

	"k8s.io/klog/v2"
)

type GlobalAuthConfig struct {
	AuthType string `gcfg:"auth-type"`
}

type AppRoleConfig struct {
	RoleId   string `gcfg:"role-id"`
	SecretId string `gcfg:"secret-id"`
}

type AuthConfig struct {
	Global  GlobalAuthConfig
	AppRole AppRoleConfig
}

type Authenticator struct {
	authConfig *AuthConfig
}

func NewAuthenticator(vaultAuthFilePath string) (*Authenticator, error) {
	cfg, err := initConfig(vaultAuthFilePath)
	if err != nil {
		return nil, err
	}

	return &Authenticator{authConfig: cfg}, nil
}

func (a *Authenticator) Authenticate(ctx context.Context, vclient *vault.Client) (*vault.Secret, error) {
	switch authType := a.authConfig.Global.AuthType; authType {
	case "approle":
		secret, err := a.approleAuthentication(ctx, vclient)
		if err != nil {
			return nil, err
		}

		return secret, nil
	default:
		return nil, fmt.Errorf("invalid Vault auth method %s", authType)
	}
}

func (a *Authenticator) approleAuthentication(ctx context.Context, vclient *vault.Client) (*vault.Secret, error) {
	secretId := auth.SecretID{
		FromString: a.authConfig.AppRole.SecretId,
	}
	approleAuth, err := auth.NewAppRoleAuth(a.authConfig.AppRole.RoleId, &secretId)
	if err != nil {
		return nil, fmt.Errorf("failed to create Vault approle authenticator")
	}

	secret, err := vclient.Auth().Login(ctx, approleAuth)
	if err != nil {
		return nil, fmt.Errorf("failed to login into Vault: %s", err)
	}

	klog.Infof("logged into Vault as %s", secret.Auth.Metadata["role_name"])

	return secret, nil
}
