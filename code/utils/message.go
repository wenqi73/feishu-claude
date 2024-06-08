package utils

import (
	"github.com/pandodao/tokenizer-go"
	"strings"
)

type Messages struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (msg *Messages) CalculateTokenLength() int {
	text := strings.TrimSpace(msg.Content)
	return tokenizer.MustCalToken(text)
}
