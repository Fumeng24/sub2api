package service

import (
	"strings"

	"github.com/tidwall/gjson"
)

func (c *openAIImageOutputCounter) addChatCompletionsChoices(choices gjson.Result) {
	if !choices.IsArray() {
		return
	}
	choices.ForEach(func(_, choice gjson.Result) bool {
		choice.Get("message.images").ForEach(func(_, image gjson.Result) bool {
			c.addChatCompletionsImageItem(image)
			return true
		})
		return true
	})
}

func (c *openAIImageOutputCounter) addChatCompletionsImageItem(item gjson.Result) {
	if !item.Exists() || !item.IsObject() {
		return
	}
	result := ""
	for _, path := range []string{
		"image_url.url",
		"url",
		"b64_json",
		"image_b64",
		"image_base64",
		"base64_json",
		"base64",
	} {
		if value := strings.TrimSpace(item.Get(path).String()); value != "" {
			result = value
			break
		}
	}
	if result == "" {
		return
	}
	key := hashOpenAIImageOutputResult(result)
	if key == "" {
		return
	}
	size := strings.TrimSpace(item.Get("size").String())
	if _, exists := c.seen[key]; exists {
		if size != "" && strings.TrimSpace(c.seenSizes[key]) == "" {
			c.seenSizes[key] = size
		}
		return
	}
	c.seen[key] = struct{}{}
	c.seenOrder = append(c.seenOrder, key)
	if size != "" {
		c.seenSizes[key] = size
	}
	c.count++
}
