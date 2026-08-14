package service

import "fmt"

const openAIImageMaxGenerationCountCustom = 4

func validateOpenAIImageGenerationCountCustom(count int) error {
	if count > openAIImageMaxGenerationCountCustom {
		return fmt.Errorf("n must be less than or equal to %d", openAIImageMaxGenerationCountCustom)
	}
	return nil
}
