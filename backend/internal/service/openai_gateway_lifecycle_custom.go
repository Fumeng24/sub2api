package service

// Start and Stop preserve the local server lifecycle contract. The upstream
// gateway currently has no background lifecycle work of its own.
func (s *OpenAIGatewayService) Start() {}

func (s *OpenAIGatewayService) Stop() {}
