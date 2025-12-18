package proxy

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

var toolUseIDCounter uint64

const geminiCLIClaudeThoughtSignature = "skip_thought_signature_validator"

// Alias2ModelName 将用户请求的模型名转换为 Antigravity 后端支持的名称
func Alias2ModelName(modelName string) string {
	switch modelName {
	// Claude 模型映射
	case "claude-opus-4-5-20251101":
		return "claude-opus-4-5-thinking"
	case "claude-sonnet-4-5-20250929":
		return "claude-sonnet-4-5"
	case "claude-haiku-4-5-20251001":
		return "claude-sonnet-4-5"
	// Gemini 模型映射
	case "gemini-2.5-computer-use-preview-10-2025":
		return "rev19-uic3-1p"
	case "gemini-3-pro-image-preview":
		return "gemini-3-pro-image"
	case "gemini-3-pro-preview":
		return "gemini-3-pro-high"
	case "gemini-claude-sonnet-4-5":
		return "claude-sonnet-4-5"
	case "gemini-claude-sonnet-4-5-thinking":
		return "claude-sonnet-4-5-thinking"
	case "gemini-claude-opus-4-5-thinking":
		return "claude-opus-4-5-thinking"
	default:
		return modelName
	}
}

// IsImageModel 判断是否为图片生成模型
func IsImageModel(modelName string) bool {
	return strings.Contains(modelName, "-image")
}

// GenerateSessionID 生成会话 ID
func GenerateSessionID() string {
	return fmt.Sprintf("-%d", time.Now().UnixNano()%9_000_000_000_000_000_000)
}

// GenerateProjectID 生成项目 ID
func GenerateProjectID() string {
	adjectives := []string{"useful", "bright", "swift", "calm", "bold"}
	nouns := []string{"fuze", "wave", "spark", "flow", "core"}
	adj := adjectives[time.Now().UnixNano()%int64(len(adjectives))]
	noun := nouns[time.Now().UnixNano()%int64(len(nouns))]
	randomPart := strings.ToLower(uuid.NewString())[:5]
	return adj + "-" + noun + "-" + randomPart
}

// GetNextToolUseID 获取下一个工具使用 ID
func GetNextToolUseID() uint64 {
	return atomic.AddUint64(&toolUseIDCounter, 1)
}

// GetThoughtSignature 获取思考签名常量
func GetThoughtSignature() string {
	return geminiCLIClaudeThoughtSignature
}

// walkJSON 递归遍历 JSON 结构，查找指定字段的所有路径
func walkJSON(value gjson.Result, path, field string, paths *[]string) {
	switch value.Type {
	case gjson.JSON:
		value.ForEach(func(key, val gjson.Result) bool {
			var childPath string
			if path == "" {
				childPath = key.String()
			} else {
				childPath = path + "." + key.String()
			}
			if key.String() == field {
				*paths = append(*paths, childPath)
			}
			walkJSON(val, childPath, field, paths)
			return true
		})
	}
}

// deleteKey 从 JSON 字符串中删除所有指定名称的字段
func deleteKey(jsonStr, keyName string) string {
	paths := make([]string, 0)
	walkJSON(gjson.Parse(jsonStr), "", keyName, &paths)
	for _, p := range paths {
		jsonStr, _ = sjson.Delete(jsonStr, p)
	}
	return jsonStr
}

// CleanUnsupportedSchemaFields 清理 Antigravity 不支持的 JSON Schema 字段
func CleanUnsupportedSchemaFields(jsonStr string) string {
	// 先处理 anyOf - 将其替换为 anyOf 数组的第一个元素
	for {
		paths := make([]string, 0)
		walkJSON(gjson.Parse(jsonStr), "", "anyOf", &paths)
		if len(paths) == 0 {
			break
		}
		for _, p := range paths {
			anyOf := gjson.Get(jsonStr, p)
			if anyOf.IsArray() {
				anyOfItems := anyOf.Array()
				if len(anyOfItems) > 0 {
					// 获取父路径
					if strings.HasSuffix(p, ".anyOf") {
						parentPath := p[:len(p)-6] // 去掉 ".anyOf"
						jsonStr, _ = sjson.SetRaw(jsonStr, parentPath, anyOfItems[0].Raw)
					} else if p == "anyOf" {
						// 顶层 anyOf - 直接删除
						jsonStr, _ = sjson.Delete(jsonStr, p)
					}
				}
			}
		}
	}

	// 然后清理不支持的字段
	unsupportedFields := []string{
		"$schema",
		"$ref",
		"$defs",
		"maxItems",
		"minItems",
		"minLength",
		"maxLength",
		"exclusiveMinimum",
		"exclusiveMaximum",
		"minimum",
		"maximum",
		"pattern",
		"format",
		"additionalProperties",
	}
	for _, field := range unsupportedFields {
		jsonStr = deleteKey(jsonStr, field)
	}

	return jsonStr
}
