package groq

import "context"

func (c *Client) ChatCompletion(
	ctx context.Context,
	req ChatCompletionRequest,
) (*ChatCompletionResponse, error) {
	var resp ChatCompletionResponse

	err := c.do(
		ctx,
		"POST",
		"/chat/completions",
		req,
		&resp,
	)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

func (c *Client) SimplePrompt(
	ctx context.Context,
	model string,
	prompt string,
	reasoningEffort string,
	temperature float64,
) (string, error) {
	resp, err := c.ChatCompletion(ctx, ChatCompletionRequest{
		Model: model,
		Messages: []ChatMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		ReasoningEffort: reasoningEffort,
		Temperature:     temperature,
	})
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", nil
	}

	return resp.Choices[0].Message.Content, nil
}
