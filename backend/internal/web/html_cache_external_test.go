//go:build !embed

package web

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExternalHTMLCache(t *testing.T) {
	cache := NewHTMLCache()
	require.Nil(t, cache.Get())

	cache.SetBaseHTML([]byte("<html></html>"))
	cache.Set([]byte("<html><body>test</body></html>"), []byte(`{"v":1}`))
	first := cache.Get()
	require.NotNil(t, first)
	require.Equal(t, []byte("<html><body>test</body></html>"), first.Content)
	require.True(t, strings.HasPrefix(first.ETag, `"`))
	require.True(t, strings.HasSuffix(first.ETag, `"`))

	cache.Invalidate()
	require.Nil(t, cache.Get())

	cache.Set([]byte("<html><body>test</body></html>"), []byte(`{"v":2}`))
	second := cache.Get()
	require.NotNil(t, second)
	require.NotEqual(t, first.ETag, second.ETag)
}
