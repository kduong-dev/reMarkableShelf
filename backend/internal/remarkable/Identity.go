package remarkable

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/ansel1/merry"
	"golang.org/x/crypto/ssh"
)

// keyComment labels the server's key in a tablet's authorized_keys, so it
// can be recognised and removed by hand.
const keyComment = "remarkable-shelf"

// LoadOrCreateSigner loads the server's SSH key from path, generating and
// saving a new ed25519 key there on first run. The key is the server's
// identity to every paired tablet, so path belongs on persistent storage.
func LoadOrCreateSigner(path string) (ssh.Signer, error) {
	pemBytes, err := os.ReadFile(path)
	if err == nil {
		signer, err := ssh.ParsePrivateKey(pemBytes)
		if err != nil {
			return nil, merry.Wrap(err).WithUserMessagef("reading SSH key %s", path)
		}
		return signer, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, merry.Wrap(err)
	}
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, merry.Wrap(err)
	}
	block, err := ssh.MarshalPrivateKey(privateKey, keyComment)
	if err != nil {
		return nil, merry.Wrap(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, merry.Wrap(err)
	}
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0o600); err != nil {
		return nil, merry.Wrap(err).WithUserMessagef("saving SSH key %s", path)
	}
	return ssh.NewSignerFromKey(privateKey)
}

// authorizedKey is the signer's public key as an authorized_keys line.
func authorizedKey(signer ssh.Signer) string {
	line := string(ssh.MarshalAuthorizedKey(signer.PublicKey()))
	return line[:len(line)-1] + " " + keyComment
}
