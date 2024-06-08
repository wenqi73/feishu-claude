package bedrock

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"log"
	"start-feishubot/initialization"
	"start-feishubot/logger"
	"start-feishubot/utils"
)

type AIMode float64

const (
	Fresh      AIMode = 0.1
	Warmth     AIMode = 0.7
	Balance    AIMode = 1.2
	Creativity AIMode = 1.7
)

var AIModeMap = map[string]AIMode{
	"严谨": Fresh,
	"简洁": Warmth,
	"标准": Balance,
	"发散": Creativity,
}

var AIModeStrs = []string{
	"严谨",
	"简洁",
	"标准",
	"发散",
}

// ClaudeResponseBody 请求体
type ClaudeResponseBody struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int                    `json:"created"`
	Model   string                 `json:"model"`
	Choices []ClaudeChoiceItem     `json:"choices"`
	Usage   map[string]interface{} `json:"usage"`
}

type ClaudeChoiceItem struct {
	Message      utils.Messages `json:"message"`
	Index        int            `json:"index"`
	FinishReason string         `json:"finish_reason"`
}

// ClaudeRequestBody 响应体
type ClaudeRequestBody struct {
	Model            string           `json:"model"`
	Messages         []utils.Messages `json:"messages"`
	MaxTokens        int              `json:"max_tokens"`
	Temperature      AIMode           `json:"temperature"`
	TopP             int              `json:"top_p"`
	FrequencyPenalty int              `json:"frequency_penalty"`
	PresencePenalty  int              `json:"presence_penalty"`
}

func (claude *Claude) Completions(msg []utils.Messages) (resp utils.Messages,
	err error) {

	var contentBlocks []types.ContentBlock
	for _, item := range msg {
		contentBlocks = append(contentBlocks, &types.ContentBlockMemberText{Value: item.Content})
	}

	messages := []types.Message{
		{
			Role:    "user",
			Content: contentBlocks,
		},
	}

	requestBody := &bedrockruntime.ConverseInput{
		Messages: messages,
		ModelId:  aws.String(initialization.GetConfig().AwsBedrockModel),
		InferenceConfig: &types.InferenceConfiguration{
			MaxTokens: aws.Int32(int32(claude.MaxTokens)),
		},
	}

	response, err := claude.sendRequestWithBodyType(requestBody)

	if err != nil {
		return
	}

	jsonData, err := json.Marshal(response.Output)
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return
	}

	var result map[string]interface{}

	// 解码 JSON 数据到 map
	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		log.Fatalf("Error occurred during unmarshaling. Error: %s", err.Error())
	}

	fmt.Printf("response is: \n %s\n", string(jsonData))

	value := result["Value"].(map[string]interface{})
	content := value["Content"].([]interface{})
	if err == nil && len(content) > 0 {
		firstContent := content[0].(map[string]interface{})
		firstValue := firstContent["Value"].(string)
		firstRole := value["Role"].(string)
		resp = utils.Messages{
			Content: firstValue,
			Role:    firstRole,
		}
	} else {
		logger.Errorf("ERROR %v", err)
		resp = utils.Messages{}
		err = errors.New("openai 请求失败")
	}
	return resp, err
}
