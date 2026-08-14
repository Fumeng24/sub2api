package service

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type geminiNativeStreamingResultWithImageAccounting struct {
	result     *geminiNativeStreamResult
	imageCount int
}

type geminiImageAccountingContextKey struct{}

type geminiImageAccountingContext struct {
	requireImage bool
	imageCount   int
}

func bindGeminiImageAccountingCustom(c *gin.Context, requireImage bool) (state *geminiImageAccountingContext, restore func()) {
	state = &geminiImageAccountingContext{requireImage: requireImage}
	if c == nil || c.Request == nil {
		return state, func() {}
	}
	originalRequest := c.Request
	c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), geminiImageAccountingContextKey{}, state))
	return state, func() { c.Request = originalRequest }
}

func geminiImageAccountingContextFromGin(c *gin.Context) (*geminiImageAccountingContext, bool) {
	if c == nil || c.Request == nil {
		return nil, false
	}
	state, ok := c.Request.Context().Value(geminiImageAccountingContextKey{}).(*geminiImageAccountingContext)
	return state, ok
}

func recordGeminiNonStreamingImageAccountingCustom(c *gin.Context, response map[string]any) error {
	state, ok := geminiImageAccountingContextFromGin(c)
	if !ok {
		return nil
	}
	state.imageCount = countGeminiImagePartsFromMap(response)
	if state.requireImage && state.imageCount == 0 {
		return newGeminiEmptyImageFailoverError()
	}
	return nil
}

func recordGeminiNativeImageAccountingCustom(c *gin.Context, body []byte) error {
	state, ok := geminiImageAccountingContextFromGin(c)
	if !ok {
		return nil
	}
	state.imageCount = countGeminiImagePartsFromBytes(body)
	if state.requireImage && state.imageCount == 0 {
		return newGeminiEmptyImageFailoverError()
	}
	return nil
}

func recordGeminiStreamingImageAccountingCustom(c *gin.Context, body []byte) {
	state, ok := geminiImageAccountingContextFromGin(c)
	if !ok {
		return
	}
	state.imageCount += countGeminiImagePartsFromBytes(body)
}

func (s *GeminiMessagesCompatService) handleNativeStreamingResponseWithImageAccounting(
	c *gin.Context,
	resp *http.Response,
	startTime time.Time,
	isOAuth bool,
) (*geminiNativeStreamingResultWithImageAccounting, error) {
	state, restore := bindGeminiImageAccountingCustom(c, false)
	defer restore()
	result, err := s.handleNativeStreamingResponse(c, resp, startTime, isOAuth)
	if err != nil {
		return nil, err
	}
	return &geminiNativeStreamingResultWithImageAccounting{
		result:     result,
		imageCount: state.imageCount,
	}, nil
}

func (s *GeminiMessagesCompatService) handleNonStreamingResponseWithImageAccounting(c *gin.Context, resp *http.Response, originalModel string, requireImage bool) (*geminiNonStreamingResult, error) {
	state, restore := bindGeminiImageAccountingCustom(c, requireImage)
	defer restore()
	usage, err := s.handleNonStreamingResponse(c, resp, originalModel)
	if err != nil {
		return nil, err
	}
	return &geminiNonStreamingResult{usage: usage, imageCount: state.imageCount}, nil
}

func (s *GeminiMessagesCompatService) handleNativeNonStreamingResponseWithImageAccounting(c *gin.Context, resp *http.Response, isOAuth, requireImage bool) (*geminiNativeNonStreamingResult, error) {
	state, restore := bindGeminiImageAccountingCustom(c, requireImage)
	defer restore()
	usage, err := s.handleNativeNonStreamingResponse(c, resp, isOAuth)
	if err != nil {
		return nil, err
	}
	return &geminiNativeNonStreamingResult{usage: usage, imageCount: state.imageCount}, nil
}
