package bedrock

import (
	"context"
	"fmt"
	aws_config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"log"
	"start-feishubot/initialization"
)

type PlatForm string

const (
	MaxRetries = 3
)

const (
	AWS PlatForm = "aws"
)

var (
	brc *bedrockruntime.Client
)

type AzureConfig struct {
	BaseURL        string
	ResourceName   string
	DeploymentName string
	ApiVersion     string
	ApiToken       string
}

type Claude struct {
	Region    string
	Model     string
	MaxTokens int
}
type requestBodyType int

const (
	jsonBody requestBodyType = iota
	formVoiceDataBody
	formPictureDataBody

	nilBody
)

//request/response model

type Request struct {
	Prompt            string   `json:"prompt"`
	MaxTokensToSample int      `json:"max_tokens_to_sample"`
	Temperature       float64  `json:"temperature,omitempty"`
	TopP              float64  `json:"top_p,omitempty"`
	TopK              int      `json:"top_k,omitempty"`
	StopSequences     []string `json:"stop_sequences,omitempty"`
}

type Response struct {
	Completion string `json:"completion"`
}

func (claude *Claude) sendRequestWithBodyType(requestBody *bedrockruntime.ConverseInput) (*bedrockruntime.ConverseOutput, error) {
	output, err := brc.Converse(context.Background(), requestBody)

	if err != nil {
		fmt.Println("Error calling Converse API:", err)
		return nil, err
	}

	return output, nil
}

func (claude *Claude) ChangeMode(model string) *Claude {
	claude.Model = model
	return claude
}

func NewClaude(config initialization.Config) *Claude {
	region := config.AwsRegion

	cfg, err := aws_config.LoadDefaultConfig(context.Background(), aws_config.WithRegion(config.AwsRegion))
	if err != nil {
		log.Fatal(err)
	}

	brc = bedrockruntime.NewFromConfig(cfg)

	return &Claude{
		Region:    region,
		Model:     config.AwsBedrockModel,
		MaxTokens: config.OpenaiMaxTokens,
	}
}
