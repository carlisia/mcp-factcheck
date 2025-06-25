package factcheck

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/sashabaranov/go-openai"
)

var openaiClient *openai.Client

func InitOpenAI() error {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("OPENAI_API_KEY environment variable is not set")
	}

	openaiClient = openai.NewClient(apiKey)
	return nil
}

func AskOpenAI(userText string, specChunks []string) (string, error) {
	combinedSpec := strings.Join(specChunks, "\n\n")

	prompt := fmt.Sprintf(`You are validating content for technical correctness against the MCP specification.

The following content section needs fact-checking:

"%s"

Compare it against this official MCP specification context:

"%s"

Does the content accurately reflect the concepts in the spec? Be detailed. Point out inaccuracies, ambiguities, or missing parts, and provide suggestions for improvement if needed.`, userText, combinedSpec)

	resp, err := openaiClient.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model: openai.GPT4, // Or gpt-4o
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
	})
	if err != nil {
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
}
