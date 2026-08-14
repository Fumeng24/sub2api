package service

type createGroupInputCustom struct {
	ForceOpenAIPriority bool
	OpenAIStableLowTTFT bool
}

type updateGroupInputCustom struct {
	ForceOpenAIPriority *bool
	OpenAIStableLowTTFT *bool
}

func applyCreateOpenAIGroupFieldsCustom(group *Group, input *CreateGroupInput) {
	if group == nil || input == nil {
		return
	}
	group.ForceOpenAIPriority = input.ForceOpenAIPriority
	group.OpenAIStableLowTTFT = input.OpenAIStableLowTTFT
}

func applyUpdateOpenAIGroupFieldsCustom(group *Group, input *UpdateGroupInput) {
	if group == nil || input == nil {
		return
	}
	if input.ForceOpenAIPriority != nil {
		group.ForceOpenAIPriority = *input.ForceOpenAIPriority
	}
	if input.OpenAIStableLowTTFT != nil {
		group.OpenAIStableLowTTFT = *input.OpenAIStableLowTTFT
	}
}

func sanitizeCustomOpenAIGroupFields(group *Group) {
	if group == nil || group.Platform == PlatformOpenAI {
		return
	}
	group.ForceOpenAIPriority = false
	group.OpenAIStableLowTTFT = false
}
