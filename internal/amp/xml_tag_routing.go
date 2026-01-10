package amp

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"

	"ampmanager/internal/translator"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

const (
	XMLTagExtractedModelKey   = "xml_tag_extracted_model"
	XMLTagExtractedLevelKey   = "xml_tag_extracted_level"
	XMLTagOriginalModelKey    = "xml_tag_original_model"
	XMLTagStripTagsKey        = "xml_tag_strip_tags"
	XMLTagDetectedFormatKey   = "xml_tag_detected_format"
)

var (
	modelTagRegex = regexp.MustCompile(`<model>\s*([^<]+?)\s*</model>`)
	levelTagRegex = regexp.MustCompile(`<level>\s*([^<]+?)\s*</level>`)
)

type XMLTagExtractionResult struct {
	Model         string
	Level         string
	OriginalModel string
	TextPath      string
	OriginalText  string
}

func XMLTagRoutingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodPost {
			c.Next()
			return
		}

		contentType := c.GetHeader("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			c.Next()
			return
		}

		path := c.Request.URL.Path
		if !isTargetPath(path) {
			c.Next()
			return
		}

		if c.Request.Body == nil || c.Request.ContentLength == 0 {
			c.Next()
			return
		}

		bodyBytes, err := io.ReadAll(io.LimitReader(c.Request.Body, 10*1024*1024))
		c.Request.Body.Close()
		if err != nil {
			log.Errorf("xml_tag_routing: failed to read body: %v", err)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			c.Next()
			return
		}

		detectedFormat := detectIncomingFormatFromBody(bodyBytes, path)
		if detectedFormat != "" {
			c.Set(XMLTagDetectedFormatKey, detectedFormat)
			log.Debugf("xml_tag_routing: detected incoming format=%s for path=%s", detectedFormat, path)
		}

		result := extractXMLTags(bodyBytes, path)

		if result.Model == "" && result.Level == "" {
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			c.Next()
			return
		}

		log.Infof("xml_tag_routing: extracted model=%q, level=%q from path=%s", result.Model, result.Level, path)

		if result.Level != "" {
			normalizedLevel := normalizeLevel(result.Level)
			c.Set(XMLTagExtractedLevelKey, normalizedLevel)
			log.Debugf("xml_tag_routing: set level=%q (normalized from %q)", normalizedLevel, result.Level)
		}

		if result.Model != "" {
			c.Set(XMLTagExtractedModelKey, result.Model)
			c.Set(XMLTagOriginalModelKey, result.OriginalModel)

			modifiedBody, err := rewriteBodyWithModel(bodyBytes, result)
			if err != nil {
				log.Warnf("xml_tag_routing: failed to rewrite body: %v", err)
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			} else {
				bodyBytes = modifiedBody
				log.Infof("xml_tag_routing: rewrote model %q -> %q", result.OriginalModel, result.Model)
			}
		}

		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		c.Request.ContentLength = int64(len(bodyBytes))

		c.Next()
	}
}

func isTargetPath(path string) bool {
	targetPaths := []string{
		"/v1/responses",
		"/v1/messages",
		"/v1/chat/completions",
		"/api/internal",
	}
	for _, tp := range targetPaths {
		if strings.HasPrefix(path, tp) {
			return true
		}
	}
	return false
}

func extractXMLTags(body []byte, path string) XMLTagExtractionResult {
	result := XMLTagExtractionResult{}

	originalModel := gjson.GetBytes(body, "model").String()
	result.OriginalModel = originalModel

	var textCandidates []string

	if strings.Contains(path, "/responses") {
		textCandidates = extractTextFromOpenAIResponses(body)
	} else if strings.Contains(path, "/messages") {
		textCandidates = extractTextFromClaudeMessages(body)
	} else if strings.Contains(path, "/chat/completions") {
		textCandidates = extractTextFromOpenAIChat(body)
	} else if strings.Contains(path, "/internal") {
		textCandidates = append(textCandidates, extractTextFromOpenAIResponses(body)...)
		textCandidates = append(textCandidates, extractTextFromClaudeMessages(body)...)
	}

	for _, text := range textCandidates {
		if result.Model == "" {
			if match := modelTagRegex.FindStringSubmatch(text); len(match) > 1 {
				result.Model = strings.TrimSpace(match[1])
				result.OriginalText = text
			}
		}
		if result.Level == "" {
			if match := levelTagRegex.FindStringSubmatch(text); len(match) > 1 {
				result.Level = strings.TrimSpace(match[1])
			}
		}
		if result.Model != "" && result.Level != "" {
			break
		}
	}

	return result
}

func extractTextFromOpenAIResponses(body []byte) []string {
	var texts []string

	input := gjson.GetBytes(body, "input")
	if !input.Exists() || !input.IsArray() {
		return texts
	}

	input.ForEach(func(_, item gjson.Result) bool {
		content := item.Get("content")
		if content.Exists() && content.IsArray() {
			content.ForEach(func(_, contentItem gjson.Result) bool {
				if contentItem.Get("type").String() == "text" {
					text := contentItem.Get("text").String()
					if text != "" {
						texts = append(texts, text)
					}
				}
				return true
			})
		}
		return true
	})

	return texts
}

func extractTextFromClaudeMessages(body []byte) []string {
	var texts []string

	messages := gjson.GetBytes(body, "messages")
	if !messages.Exists() || !messages.IsArray() {
		return texts
	}

	messages.ForEach(func(_, msg gjson.Result) bool {
		content := msg.Get("content")
		if content.Exists() {
			if content.IsArray() {
				content.ForEach(func(_, contentItem gjson.Result) bool {
					if contentItem.Get("type").String() == "text" {
						text := contentItem.Get("text").String()
						if text != "" {
							texts = append(texts, text)
						}
					}
					return true
				})
			} else if content.Type == gjson.String {
				texts = append(texts, content.String())
			}
		}
		return true
	})

	return texts
}

func extractTextFromOpenAIChat(body []byte) []string {
	var texts []string

	messages := gjson.GetBytes(body, "messages")
	if !messages.Exists() || !messages.IsArray() {
		return texts
	}

	messages.ForEach(func(_, msg gjson.Result) bool {
		content := msg.Get("content")
		if content.Exists() {
			if content.IsArray() {
				content.ForEach(func(_, contentItem gjson.Result) bool {
					if contentItem.Get("type").String() == "text" {
						text := contentItem.Get("text").String()
						if text != "" {
							texts = append(texts, text)
						}
					}
					return true
				})
			} else if content.Type == gjson.String {
				texts = append(texts, content.String())
			}
		}
		return true
	})

	return texts
}

func rewriteBodyWithModel(body []byte, result XMLTagExtractionResult) ([]byte, error) {
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	payload["model"] = result.Model

	stripTagsFromPayload(payload, result)

	return json.Marshal(payload)
}

func stripTagsFromPayload(payload map[string]interface{}, result XMLTagExtractionResult) {
	if input, ok := payload["input"].([]interface{}); ok {
		for _, item := range input {
			if itemMap, ok := item.(map[string]interface{}); ok {
				stripTagsFromContent(itemMap)
			}
		}
	}

	if messages, ok := payload["messages"].([]interface{}); ok {
		for _, msg := range messages {
			if msgMap, ok := msg.(map[string]interface{}); ok {
				stripTagsFromContent(msgMap)
			}
		}
	}
}

func stripTagsFromContent(container map[string]interface{}) {
	content, ok := container["content"]
	if !ok {
		return
	}

	switch c := content.(type) {
	case []interface{}:
		for _, item := range c {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if itemMap["type"] == "text" {
					if text, ok := itemMap["text"].(string); ok {
						itemMap["text"] = stripXMLTags(text)
					}
				}
			}
		}
	case string:
		container["content"] = stripXMLTags(c)
	}
}

func stripXMLTags(text string) string {
	text = modelTagRegex.ReplaceAllString(text, "")
	text = levelTagRegex.ReplaceAllString(text, "")
	text = strings.TrimLeft(text, "\n\r\t ")
	return text
}

func normalizeLevel(level string) string {
	level = strings.ToLower(strings.TrimSpace(level))
	level = strings.ReplaceAll(level, "-", "")
	level = strings.ReplaceAll(level, "_", "")

	switch level {
	case "xhigh", "extrahigh", "veryhigh", "max":
		return "xhigh"
	case "high":
		return "high"
	case "medium", "med", "mid":
		return "medium"
	case "low", "min":
		return "low"
	case "none", "off", "disabled":
		return "none"
	default:
		return level
	}
}

func GetXMLTagExtractedLevel(c *gin.Context) string {
	if level, exists := c.Get(XMLTagExtractedLevelKey); exists {
		return level.(string)
	}
	return ""
}

func GetXMLTagExtractedModel(c *gin.Context) string {
	if model, exists := c.Get(XMLTagExtractedModelKey); exists {
		return model.(string)
	}
	return ""
}

func GetXMLTagDetectedFormat(c *gin.Context) translator.Format {
	if format, exists := c.Get(XMLTagDetectedFormatKey); exists {
		return format.(translator.Format)
	}
	return ""
}

func detectIncomingFormatFromBody(body []byte, path string) translator.Format {
	if !strings.Contains(path, "/api/internal") {
		return ""
	}

	hasInput := gjson.GetBytes(body, "input").Exists()
	hasMessages := gjson.GetBytes(body, "messages").Exists()

	if hasInput {
		inputArr := gjson.GetBytes(body, "input")
		if inputArr.IsArray() {
			for _, item := range inputArr.Array() {
				if item.Get("content").IsArray() {
					return translator.FormatOpenAIResponses
				}
			}
		}
		return translator.FormatOpenAIResponses
	}

	if hasMessages {
		firstMsg := gjson.GetBytes(body, "messages.0")
		if firstMsg.Exists() {
			role := firstMsg.Get("role").String()
			content := firstMsg.Get("content")

			if content.IsArray() {
				for _, item := range content.Array() {
					itemType := item.Get("type").String()
					if itemType == "text" || itemType == "image" || itemType == "tool_use" || itemType == "tool_result" {
						return translator.FormatClaude
					}
				}
			}

			if role == "user" || role == "assistant" || role == "system" {
				if content.IsArray() {
					return translator.FormatClaude
				}
				return translator.FormatOpenAIChat
			}
		}
	}

	return ""
}
