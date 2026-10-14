package types

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestErrorCodesStartAtTwo(t *testing.T) {
	require.Equal(t, uint32(2), ErrInvalidDenom.ABCICode())
	require.Equal(t, uint32(19), ErrCorruptOwnerIndex.ABCICode())
}
