package main

import (
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

func token(args []string) error {
	var (
		key    interface{}
		err    error
		method jwt.SigningMethod
	)

	cmd := flag.NewFlagSet("token", flag.ExitOnError)
	name := cmd.String("name", "", "Name of the token")
	namespace := cmd.String("namespace", "", "Namespace to inject resource")
	signingKeyPath := cmd.String("signing-key", "", "Path to PEM encoded signing key file")

	cmd.Parse(args)

	if *name == "" {
		return errors.New("-name must be specified")
	}
	if *namespace == "" {
		return errors.New("-namespace must be specified")
	}
	if *signingKeyPath == "" {
		return errors.New("-signing-key must be specified")
	}

	buf, err := ioutil.ReadFile(*signingKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read signing key: %v", err)
	}

	block, _ := pem.Decode(buf)
	switch block.Type {
	case "RSA PRIVATE KEY":
		method = jwt.SigningMethodRS256
		key, err = jwt.ParseRSAPrivateKeyFromPEM(buf)
	case "EC PRIVATE KEY":
		method = jwt.SigningMethodES256
		key, err = jwt.ParseECPrivateKeyFromPEM(buf)
	default:
		err = fmt.Errorf("unsupported signing key type: %v", block.Type)
	}

	if err != nil {
		return fmt.Errorf("failed to parse signing key: %v", err)
	}

	token := jwt.NewWithClaims(method, jwt.MapClaims{
		"name":      *name,
		"namespace": *namespace,
		"iat":       time.Now().Unix(),
	})

	t, err := token.SignedString(key)
	if err != nil {
		return fmt.Errorf("failed to sign token: %v", err)
	}

	fmt.Println(t)

	return nil
}
