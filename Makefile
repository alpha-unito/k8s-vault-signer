LDFLAGS := "-w -s -X 'github.com/alpha-unito/k8s-vault-signer/pkg/version.Version=$(VERSION)'"
SOURCES := Makefile go.mod go.sum $(shell find . -name '*.go' 2>/dev/null)
VERSION := "0.0.1"

docker:
	docker build -t alphaunito/k8s-vault-signer:$(VERSION) ./

build: $(SOURCES)
	CGO_ENABLED=0 go build -trimpath -ldflags $(LDFLAGS) -o vault-signer cmd/main.go

version:
	@echo $(VERSION)