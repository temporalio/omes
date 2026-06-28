package workers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBuildDirNameIsUniquePerEnvironment(t *testing.T) {
	createdAt := time.UnixMilli(123)
	first := &TestEnvironment{createdAt: createdAt}
	second := &TestEnvironment{createdAt: createdAt}

	require.Equal(t, first.buildDirName(), first.buildDirName())
	require.NotEqual(t, first.buildDirName(), second.buildDirName())
}
