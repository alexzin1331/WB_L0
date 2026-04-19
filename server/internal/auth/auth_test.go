package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManagerGenerateAndParse(t *testing.T) {
	manager := NewManager("test-secret", time.Hour)

	token, err := manager.Generate(7, "alice")
	require.NoError(t, err)

	claims, err := manager.Parse(token)
	require.NoError(t, err)
	assert.Equal(t, int64(7), claims.UserID)
	assert.Equal(t, "alice", claims.Username)
}

func TestManagerRejectsInvalidSignature(t *testing.T) {
	manager := NewManager("test-secret", time.Hour)

	token, err := manager.Generate(7, "alice")
	require.NoError(t, err)

	_, err = NewManager("other-secret", time.Hour).Parse(token)
	require.ErrorIs(t, err, ErrInvalidToken)
}

func TestPasswordHashAndCheck(t *testing.T) {
	hash, err := HashPassword("secret123")
	require.NoError(t, err)

	require.NoError(t, CheckPassword(hash, "secret123"))
	require.Error(t, CheckPassword(hash, "wrong-password"))
}
