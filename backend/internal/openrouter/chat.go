package openrouter

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
	topP *float64,
	maxCompletionTokens int,
) (string, Usage, error) {
	resp, err := c.ChatCompletion(ctx, ChatCompletionRequest{
		Model: model,
		Messages: []ChatMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		ReasoningEffort:     reasoningEffort,
		Temperature:         temperature,
		TopP:                topP,
		MaxCompletionTokens: maxCompletionTokens,
	})
	if err != nil {
		return "", Usage{}, err
	}

	if len(resp.Choices) == 0 {
		return "", resp.Usage, nil
	}

	return resp.Choices[0].Message.Content, resp.Usage, nil
}
