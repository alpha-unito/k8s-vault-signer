# Kubernetes CSR Signer for Vault

HashCorp [Vault](https://www.vaultproject.io/) can be used as a Certificate Authority manager for [Kubernetes](https://kubernetes.io/). However, the CA key cannot be extracted from Vault, preventing the [control plane signer](https://kubernetes.io/docs/reference/access-authn-authz/certificate-signing-requests/#signer-control-plane) from properly handle Certificate Signing Request (CSR) objects in Kubernetes.

This repository implements a custom Kubernetes signer, called `unito.it/vault-signer`, to automatically handle the CSRs signing process in Kubernetes. 