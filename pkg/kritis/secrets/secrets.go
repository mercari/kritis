/*
Copyright 2018 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package secrets

import (
	"encoding/base64"
	"fmt"

	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kubernetesutil "github.com/grafeas/kritis/pkg/kritis/kubernetes"
	"github.com/pkg/errors"
)

const (
	// Public Key constant for Attestation Secrets.
	PrivateKey = "private"
	// Private Key constant for Attestation Secrets.
	PublicKey = "public"
	// Passphrase constant for Attestation Secrets.
	Passphrase = "passphrase"
)

var (
	// For testing
	getSecretFunc = getSecret
)

// PGPSigningSecret represents gpg private/public key pair secret in your
// kubernetes cluster, where private key was decrypted with the passphrase.
// The secret expects private and public key to be stored in "private" and
// "public" keys, and private key to be decrypted with the "passphrase" key e.g.
// kubectl create secret generic my-secret --from-file=public=pub.gpg \
// --from-file=private=priv.key --from-literal=passphrase=<value>
type PGPSigningSecret struct {
	PgpKey     *PgpKey
	SecretName string
}

// Fetcher is the function used to fetch kubernetes secret.
type Fetcher func(namespace string, name string) (*PGPSigningSecret, error)

// Fetch fetches kubernetes secret
func Fetch(namespace string, name string) (*PGPSigningSecret, error) {
	secret, err := getSecretFunc(namespace, name)
	if err != nil {
		return nil, err
	}
	pub, ok := secret.Data[PublicKey]
	if !ok {
		return nil, fmt.Errorf("invalid secret %s. could not find key %s", name, PublicKey)
	}
	priv, ok := secret.Data[PrivateKey]
	if !ok {
		return nil, fmt.Errorf("invalid secret %s. could not find key %s", name, PrivateKey)
	}
	pb, ok := secret.Data[Passphrase]
	phrase := ""
	if ok {
		// Passphrase was provided
		// Verify the passphrase is base64 encoded
		decoded := make([]byte, base64.StdEncoding.DecodedLen(len(pb)))
		decLen, err := base64.StdEncoding.Decode(decoded, pb)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode base64")
		}
		phrase = string(decoded[:decLen])
	}
	pgpKey, err := NewPgpKey(string(priv), phrase, string(pub))
	if err != nil {
		return nil, err
	}
	return &PGPSigningSecret{
		PgpKey:     pgpKey,
		SecretName: secret.Name,
	}, nil
}

func getSecret(namespace string, name string) (*v1.Secret, error) {
	c, err := kubernetesutil.GetClientset()
	if err != nil {
		return nil, err
	}
	return c.CoreV1().Secrets(namespace).Get(name, meta_v1.GetOptions{})
}
