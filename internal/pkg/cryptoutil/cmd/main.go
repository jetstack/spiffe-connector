package main

import (
	"fmt"
	"github.com/jetstack/spiffe-connector/internal/pkg/cryptoutil"
	"os"
)

const testConfig = ``

// Generate testing material for use locally.
func main() {
	certs, err := cryptoutil.SelfSignedServingCert()
	exitOnErr(err)

}

func exitOnErr(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't generate certs")
		os.Exit(1)
	}
}
