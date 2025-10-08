package secret

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSealer(t *testing.T) {
	secret := "seCRet"
	s := New(secret)

	id := uuid.NewString()
	sealedText := s.Seal([]byte(id))

	t.Logf("id: %s", id)
	t.Logf("sealed: %s", sealedText)

	s1 := New(secret)
	id1, err := s1.Unseal(sealedText)

	assert.NoError(t, err, "unexpected error")

	idUnsealed := string(id1)

	t.Logf("id unsealed: %s", idUnsealed)
}
