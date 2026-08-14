package apicompat

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWebSearchActionUnmarshalAcceptsObjectStringAndNull(t *testing.T) {
	t.Run("object", func(t *testing.T) {
		var out ResponsesOutput
		require.NoError(t, json.Unmarshal([]byte(`{"type":"web_search_call","action":{"type":"search","query":"latest news"}}`), &out))
		require.NotNil(t, out.Action)
		require.Equal(t, "search", out.Action.Type)
		require.Equal(t, "latest news", out.Action.Query)
	})

	t.Run("string", func(t *testing.T) {
		var out ResponsesOutput
		require.NoError(t, json.Unmarshal([]byte(`{"type":"web_search_call","action":"search"}`), &out))
		require.NotNil(t, out.Action)
		require.Equal(t, "search", out.Action.Type)
		require.Empty(t, out.Action.Query)
	})

	t.Run("null", func(t *testing.T) {
		var out ResponsesOutput
		require.NoError(t, json.Unmarshal([]byte(`{"type":"web_search_call","action":null}`), &out))
		require.Nil(t, out.Action)
	})
}
