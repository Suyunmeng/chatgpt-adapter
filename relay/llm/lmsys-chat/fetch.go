package lmsys_chat

import (
	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/logger"
	"context"
	"github.com/bincooo/emit.io"
	"github.com/google/uuid"
	"github.com/iocgo/sdk/env"
	"net/http"
	"strings"
	"sync"
)

const (
	baseUrl = "https://lmarena.ai"
)

var (
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36 Edg/124.0.0.0"
	clearance = ""
	lang      = ""

	mu    sync.Mutex
	state int32 = 0 // 0 常态 1 等待中
)

type LmsysChatRequest struct {
	Id              string `json:"id"`
	Mode            string `json:"mode"`
	ModelAId        string `json:"modelAId"`
	UserMessageId   string `json:"userMessageId"`
	ModelAMessageId string `json:"modelAMessageId"`
	Modality        string `json:"modality"`

	Messages []LmsysChatMessage `json:"messages"`
}

type LmsysChatMessage struct {
	Id                      string        `json:"id"`
	Role                    string        `json:"role"`
	Content                 string        `json:"content"`
	ExperimentalAttachments []interface{} `json:"experimental_attachments"`
	ParentMessageIds        []string      `json:"parentMessageIds"`
	ParticipantPosition     string        `json:"participantPosition"`
	ModelId                 *string       `json:"modelId"`
	EvaluationSessionId     string        `json:"evaluationSessionId"`
	Status                  string        `json:"status"`
	FailureReason           interface{}   `json:"failureReason"`
}

func fetch(ctx context.Context, cookie string, messages, modelId string) (response *http.Response, err error) {

	// 获取 cf_bm cookie 配置并合并到请求 cookie 中
	cfBmCookie := env.Env.GetString("lmsys-chat.cf_bm")
	if cfBmCookie != "" {
		if cookie != "" {
			cookie = cookie + "; " + cfBmCookie
		} else {
			cookie = cfBmCookie
		}
	}

	sessionId := uuid.NewString()
	messageId := uuid.NewString()
	modelMessageId := uuid.NewString()

	req := LmsysChatRequest{
		Id:              uuid.NewString(),
		Mode:            "direct",
		ModelAId:        modelId,
		UserMessageId:   messageId,
		ModelAMessageId: modelMessageId,
		Messages: []LmsysChatMessage{
			{
				Id:                      messageId,
				Role:                    "user",
				Content:                 messages,
				ExperimentalAttachments: make([]interface{}, 0),
				ParentMessageIds:        make([]string, 0),
				ParticipantPosition:     "a",
				ModelId:                 nil,
				EvaluationSessionId:     sessionId,
				Status:                  "pending",
				FailureReason:           nil,
			},
			{
				Id:                      modelMessageId,
				Role:                    "assistant",
				Content:                 "",
				ExperimentalAttachments: make([]interface{}, 0),
				ParentMessageIds: []string{
					messageId,
				},
				ParticipantPosition: "a",
				ModelId:             &modelId,
				EvaluationSessionId: sessionId,
				Status:              "pending",
				FailureReason:       nil,
			},
		},
		Modality: "chat",
	}

	response, err = emit.ClientBuilder(common.HTTPClient).
		Context(ctx).
		Header("User-Agent", userAgent).
		Header("Accept-Language", "en-US,en;q=0.5").
		Header("Cache-Control", "no-cache").
		Header("Accept-Encoding", "gzip, deflate, br, zstd").
		Header("Origin", baseUrl).
		Header("Cookie", cookie).
		Ja3().
		JSONHeader().
		POST(baseUrl+"/api/stream/create-evaluation").
		Body(req).
		DoC(emit.Status(http.StatusOK), emit.IsSTREAM)
	
	// 检查响应头中是否有新的 cf_bm cookie 并更新配置
	if err == nil && response != nil {
		updateCfBmCookie(response)
	}
	
	return
}

// updateCfBmCookie 检查响应头中的 Set-Cookie，如果包含 __cf_bm，则更新配置
func updateCfBmCookie(response *http.Response) {
	setCookies := response.Header.Values("Set-Cookie")
	for _, cookieStr := range setCookies {
		if strings.Contains(cookieStr, "__cf_bm=") {
			// 提取 __cf_bm cookie 值
			parts := strings.Split(cookieStr, ";")
			if len(parts) > 0 {
				cfBmPart := strings.TrimSpace(parts[0])
				if strings.HasPrefix(cfBmPart, "__cf_bm=") {
					// 更新环境变量中的 cf_bm 配置
					env.Env.Set("lmsys-chat.cf_bm", cfBmPart)
					logger.Infof("Updated lmsys-chat cf_bm cookie: %s", cfBmPart)
					break
				}
			}
		}
	}
}
