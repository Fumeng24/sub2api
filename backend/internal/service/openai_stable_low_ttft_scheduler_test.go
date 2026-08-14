package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func stableLowTTFTCandidate(id int64, score, ttft float64, measured bool) openAIAccountCandidateScore {
	return openAIAccountCandidateScore{
		account:  &Account{ID: id, Priority: int(id)},
		loadInfo: &AccountLoadInfo{AccountID: id},
		score:    score,
		ttft:     ttft,
		hasTTFT:  measured,
	}
}

func stableLowTTFTCandidateIDs(candidates []openAIAccountCandidateScore) []int64 {
	ids := make([]int64, 0, len(candidates))
	for _, candidate := range candidates {
		ids = append(ids, candidate.account.ID)
	}
	return ids
}

func TestOpenAIStableLowTTFTGroupDetection(t *testing.T) {
	require.True(t, isOpenAIStableLowTTFTGroup(&Group{Platform: PlatformOpenAI, OpenAIStableLowTTFT: true}))
	require.False(t, isOpenAIStableLowTTFTGroup(&Group{Platform: PlatformOpenAI}))
	require.False(t, isOpenAIStableLowTTFTGroup(&Group{Platform: PlatformAnthropic, OpenAIStableLowTTFT: true}))
	require.False(t, isOpenAIStableLowTTFTGroup(nil))
}

func TestSelectTopKOpenAIStableLowTTFTCandidatesPrefersMeasuredLatency(t *testing.T) {
	candidates := []openAIAccountCandidateScore{
		stableLowTTFTCandidate(1, 100, 2400, true),
		stableLowTTFTCandidate(2, 1, 350, true),
		stableLowTTFTCandidate(3, 50, 900, true),
		stableLowTTFTCandidate(4, 200, 0, false),
	}

	selected := selectTopKOpenAIStableLowTTFTCandidates(candidates, 2, 1)
	require.Equal(t, []int64{2, 3}, stableLowTTFTCandidateIDs(selected))
}

func TestSelectTopKOpenAIStableLowTTFTCandidatesWarmsUnknownAccounts(t *testing.T) {
	candidates := []openAIAccountCandidateScore{
		stableLowTTFTCandidate(1, 100, 300, true),
		stableLowTTFTCandidate(2, 90, 400, true),
		stableLowTTFTCandidate(3, 80, 500, true),
		stableLowTTFTCandidate(4, 70, 0, false),
		stableLowTTFTCandidate(5, 60, 0, false),
	}

	normal := selectTopKOpenAIStableLowTTFTCandidates(candidates, 3, 99)
	probeOne := selectTopKOpenAIStableLowTTFTCandidates(candidates, 3, 100)
	probeTwo := selectTopKOpenAIStableLowTTFTCandidates(candidates, 3, 200)

	require.Equal(t, []int64{1, 2, 3}, stableLowTTFTCandidateIDs(normal))
	require.Equal(t, []int64{1, 2, 4}, stableLowTTFTCandidateIDs(probeOne))
	require.Equal(t, []int64{1, 2, 5}, stableLowTTFTCandidateIDs(probeTwo))
}
