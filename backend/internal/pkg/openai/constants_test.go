package openai

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultModelsIncludeBareGPT56Alias(t *testing.T) {
	require.Contains(t, DefaultModelIDs(), "gpt-5.6")
}

func TestDefaultModelsPreferConcreteGPT56SolOverBareAlias(t *testing.T) {
	ids := DefaultModelIDs()
	solIndex, aliasIndex := -1, -1
	for index, id := range ids {
		switch id {
		case "gpt-5.6-sol":
			solIndex = index
		case "gpt-5.6":
			aliasIndex = index
		}
	}
	require.GreaterOrEqual(t, solIndex, 0)
	require.GreaterOrEqual(t, aliasIndex, 0)
	require.Less(t, solIndex, aliasIndex)
}
