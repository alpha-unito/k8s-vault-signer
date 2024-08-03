# Kubernetes CSR Signer for Vault

HashCorp [Vault](https://www.vaultproject.io/) can be used as a Certificate Authority manager for [Kubernetes](https://kubernetes.io/). However, the CA key cannot be extracted from Vault, preventing the [control plane signer](https://kubernetes.io/docs/reference/access-authn-authz/certificate-signing-requests/#signer-control-plane) from properly handle Certificate Signing Request (CSR) objects in Kubernetes.

This repository implements a custom Kubernetes signer, called `unito.it/vault-signer`, to automatically handle the CSRs signing process in Kubernetes.

## Usage

### Configure the Vault PKI

Vault can be configured to serve as a Certificate Authority (CA) for Kubernetes through the PKI secret engine (see [here](https://developer.hashicorp.com/vault/tutorials/secrets-management/pki-engine)). Let's assume that there exists a `pki/` mount point that contains the Kubernetes CA, which should be used to sign Kubernetes CSRs.

The most secure way to sign CSRs in a controlled way is through the `pki/sign/:role` endpoint. Therefore, it is necessary to create a role for the Kubernetes signer

```bash
vault write pki/roles/kubernetes-signer \
  allow_any_name=true                   \
  allow_glob_domains=true               \
  enforce_hostnames=false               \
  key_bits=2048                         \
  key_type=any                          \
  no_store=true                         \
  ttl="0s"                              \
  max_ttl="87600h"
```

The Vault signer needs to read the role characteristics for CSR validation and to sign CSRs by sending requests to the `pki/sign/kubernetes-signer` endpoint. Therefore, it is necessary to create a Vault [Policy](https://developer.hashicorp.com/vault/docs/concepts/policies) with the proper permissions

```bash
vault policy write kubernetes-signer-policy - << EOF
path "pki/roles/kubernetes-signer" {
  capabilities = ["read"]
}

path "pki/sign/kubernetes-signer" {
  capabilities = ["create", "patch", "update"]
}
EOF
```

### Authenticate with Vault

The Kubernetes Vault signer needs to authenticate with a Vault instance (or cluster) to delegate CSR signing. In detail, it supports two possible authentication methods: `approle` and `kubernetes`.

#### AppRole Authentication

The Vault [AppRole](https://developer.hashicorp.com/vault/docs/auth/approle) authentication method allows machines or apps to authenticate with Vault-defined roles. This authentication method can be enabled through the following command

```bash
vault auth enable approle
```

Then, it is necessary to create a Vault role for the Kubernetes signer

```bash
vault write auth/approle/role/kubernetes-signer  \
  policies="kubernetes-signer-policy"            \
  token_max_ttl="10m"                            \
  token_ttl="60s"
```

To authenticate with the AppRole authentication method, a client needs to know the `role id` and the associated `secret id`. These ids can be obtained through the following commands

```bash
ROLE_ID=$(vault read auth/approle/role/kubernetes-signer/role-id)
SECRET_ID=$(vault write -f auth/approle/role/kubernetes-signer/secret-id)
```

Then, create an `auth.conf` configuration file for the Vault Kubernetes signer with the following syntax, substituting the `${ROLE_ID}` and `${SECRET_ID}` placeholders with the values returned by the previos commands

```ini
[Global]
auth-type = approle

[AppRole]
role-id = ${ROLE_ID}
secret-id = ${SECRET_ID}
```

Finally, create a secret from the previous file

```kubectl
kubectl create secret generic   \
    --namespace=vault-signer    \
    --from-file=auth.conf       \
    vault-auth-config
```

#### Kubernetes Authentication

The Vault [Kubernetes](https://developer.hashicorp.com/vault/docs/auth/kubernetes) authentication can be used to authenticate with Vault using a Kubernetes `ServiceAccount` token. This authentication method can be enabled through the following command

```bash
vault auth enable kubernetes
```

Then, it is necessary to register the `ServiceAccount` JWT token and the Kubernetes cluster to Vault using this command, where the value of the `issuer` field should be equal to the value of the `--service-account-issuer` option of the `kube-apiserver` command running on the Kubernetes cluster control plane

```bash
vault write auth/kubernetes/config                \
  kubernetes_host=<Kubernetes hostname and port>  \
  kubernetes_ca_cert=<PEM-encoded Kubernetes CA>  \
  issuer=<ServiceAccount issuer>
```

At this point, it is necessary to create a Vault role for the Kubernetes signer

```bash
vault write auth/kubernetes/role/kubernetes-signer  \
  bound_service_account_names="vault-signer"        \
  bound_service_account_namespaces="vault-signer"   \
  policies="kubernetes-signer-policy"               \
  token_max_ttl="10m"                               \
  token_ttl="60s"
```

Then, create an `auth.conf` configuration file for the Vault Kubernetes signer with the following syntax

```ini
[Global]
auth-type = kubernetes

[Kubernetes]
role-name = kubernetes-signer
```

Finally, create a secret from the previous file

```kubectl
kubectl create secret generic   \
    --namespace=vault-signer    \
    --from-file=auth.conf       \
    vault-auth-config
```

### Deploy the Vault Signer

The Kubernetes Vault signer can be deployed using the [Helm](https://helm.sh/) Chart provided in the `helm` folder of this repository. First, create a `vaules.yml` file

```yaml
replicaCount: 3
vault:
  address:
    scheme: http
    hostname: <Your Vault Hostname>
    port: 8200
  auth:
    secretName: vault-auth-config
    secretKey: auth.conf
  pki: pki
  role: kubernetes-signer
```

Then, deploy a Helm release with the following command

```bash
helm install --namespace vault-signer --values values.yaml vault-signer ./helm
```

### Sign Kubernetes CSRs 

The Vault signer handles CSRs that specify a `signerName` equal to `unito.it/vault-signer`. To test that everything works properly, create a `csr.yaml` file with the following content

```yaml
apiVersion: certificates.k8s.io/v1
kind: CertificateSigningRequest
metadata:
  name: vault-test-csr
spec:
  request: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURSBSRVFVRVNULS0tLS0KTUlJQ1ZqQ0NBVDRDQVFBd0VURVBNQTBHQTFVRUF3d0dZVzVuWld4aE1JSUJJakFOQmdrcWhraUc5dzBCQVFFRgpBQU9DQVE4QU1JSUJDZ0tDQVFFQTByczhJTHRHdTYxakx2dHhWTTJSVlRWMDNHWlJTWWw0dWluVWo4RElaWjBOCnR2MUZtRVFSd3VoaUZsOFEzcWl0Qm0wMUFSMkNJVXBGd2ZzSjZ4MXF3ckJzVkhZbGlBNVhwRVpZM3ExcGswSDQKM3Z3aGJlK1o2MVNrVHF5SVBYUUwrTWM5T1Nsbm0xb0R2N0NtSkZNMUlMRVI3QTVGZnZKOEdFRjJ6dHBoaUlFMwpub1dtdHNZb3JuT2wzc2lHQ2ZGZzR4Zmd4eW8ybmlneFNVekl1bXNnVm9PM2ttT0x1RVF6cXpkakJ3TFJXbWlECklmMXBMWnoyalVnald4UkhCM1gyWnVVV1d1T09PZnpXM01LaE8ybHEvZi9DdS8wYk83c0x0MCt3U2ZMSU91TFcKcW90blZtRmxMMytqTy82WDNDKzBERHk5aUtwbXJjVDBnWGZLemE1dHJRSURBUUFCb0FBd0RRWUpLb1pJaHZjTgpBUUVMQlFBRGdnRUJBR05WdmVIOGR4ZzNvK21VeVRkbmFjVmQ1N24zSkExdnZEU1JWREkyQTZ1eXN3ZFp1L1BVCkkwZXpZWFV0RVNnSk1IRmQycVVNMjNuNVJsSXJ3R0xuUXFISUh5VStWWHhsdnZsRnpNOVpEWllSTmU3QlJvYXgKQVlEdUI5STZXT3FYbkFvczFqRmxNUG5NbFpqdU5kSGxpT1BjTU1oNndLaTZzZFhpVStHYTJ2RUVLY01jSVUyRgpvU2djUWdMYTk0aEpacGk3ZnNMdm1OQUxoT045UHdNMGM1dVJVejV4T0dGMUtCbWRSeEgvbUNOS2JKYjFRQm1HCkkwYitEUEdaTktXTU0xMzhIQXdoV0tkNjVoVHdYOWl4V3ZHMkh4TG1WQzg0L1BHT0tWQW9FNkpsYWFHdTlQVmkKdjlOSjVaZlZrcXdCd0hKbzZXdk9xVlA3SVFjZmg3d0drWm89Ci0tLS0tRU5EIENFUlRJRklDQVRFIFJFUVVFU1QtLS0tLQo=
  signerName: unito.it/vault-signer
  expirationSeconds: 86400
  usages:
    - client auth
```

Then, create the Kubernetes CSR object and approve it using the following commands

```bash
kubectl apply -f csr.yaml
kubectl certificate approve vault-test-csr
```

At this point, the CSR should be in the `Approved,Issued` status when inspected using the following command

```bash
kubectl get csr vault-test-csr
```

Plus, the following command should display a valid X509 certificate

```bash
kubectl get csr vault-test-csr --output=json |  \
  jq .status.certificate |                      \
  tr -d '"' |                                   \
  base64 -d |                                   \
  openssl x509 -text -noout
```

## Acknowledgment

The development of the Kubernetes Vault signer has been partially supported by the [HaMMon](https://www.supercomputing-icsc.it/en/2023/11/02/the-hammon-project-for-the-assessment-of-risks-related-to-extreme-climatic-events/) project, "Hazard Mapping and Vulnerability Monitoring", funded by the Italian Research Center in High-Performance Computing, Big Data, and Quantum Computing (ICSC).
