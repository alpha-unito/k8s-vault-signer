package version

import (
	"fmt"
	"os"
)

var Version string

func PrintVersionAndExit() {
	fmt.Printf("vault-signer v%s\n", Version)
	os.Exit(0)
}
