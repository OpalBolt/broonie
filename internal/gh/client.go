package gh

import (
	"fmt"

	"github.com/OpalBolt/broonie/internal/crypto"
	"github.com/OpalBolt/broonie/internal/db"
	"github.com/google/go-github/v62/github"
)

// NewClient decrypts the repo's stored token and returns an authenticated GitHub client.
func NewClient(repo db.Repo, encryptionKey [32]byte) (*github.Client, error) {
	if len(repo.TokenEnc) == 0 {
		return nil, fmt.Errorf("repo %s/%s has no stored token", repo.Owner, repo.Name)
	}
	token, err := crypto.Decrypt(repo.TokenEnc, encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt token for %s/%s: %w", repo.Owner, repo.Name, err)
	}
	return github.NewClient(nil).WithAuthToken(string(token)), nil
}
