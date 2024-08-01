FROM golang:1.22.5 AS builder

COPY Makefile go.mod go.sum /build/
COPY cmd/ /build/cmd/
COPY internal/ /build/internal/
COPY pkg/ /build/pkg/

RUN make -C /build/ build


FROM scratch

LABEL maintainer="iacopo.colonnelli@unito.it"

COPY --from=builder /build/vault-signer /bin/vault-signer

ENTRYPOINT [ "/bin/vault-signer" ]