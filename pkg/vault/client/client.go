package client

import (
	"os"

	"github.com/hashicorp/vault/api"
	"gopkg.in/gcfg.v1"
)

func NewClient(address string) (*api.Client, error) {
	vclient, err := api.NewClient(&api.Config{
		Address:    address,
		MaxRetries: 10,
	})
	if err != nil {
		return nil, err
	}

	return vclient, nil
}

func initConfig(configFilePath string) (*AuthConfig, error) {
	config, err := os.Open(configFilePath)
	defer func() { _ = config.Close() }()
	if err != nil {
		return nil, err
	}

	cfg := AuthConfig{}
	err = gcfg.FatalOnly(gcfg.ReadInto(&cfg, config))
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
