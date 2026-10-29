package openai

import (
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"

	"github.com/openai/openai-go/v3/option"
	ai "github.com/pchkauu/want-keep/backend/internal/ai/domain"
	receiptcontract "github.com/pchkauu/want-keep/backend/internal/receipts/contract"
)

//go:embed receipt_contract.json
var receiptContractJSON []byte

func (c *Client) ReceiptPages() (*Client, error) {
	var contract runtimeContract
	if err := json.Unmarshal(receiptContractJSON, &contract); err != nil {
		return nil, err
	}
	if contract.RuntimeSchemaAdaptation != "receipt_extraction_v1" || contract.Model != c.contract.Model || contract.ProductionAdmitted || contract.metadata().Validate() != nil {
		return nil, ai.ErrInvalidAttempt
	}
	promptHash := sha256.Sum256([]byte(contract.Prompt))
	if hex.EncodeToString(promptHash[:]) != contract.PromptFingerprint {
		return nil, ai.ErrInvalidAttempt
	}
	return &Client{sdk: c.sdk, contract: contract}, nil
}

type imageContent struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
	Detail   string `json:"detail,omitempty"`
}
type imageMessage struct {
	Role    string         `json:"role"`
	Content []imageContent `json:"content"`
}

func receiptInput(request ai.Request) []option.RequestOption {
	if request.Purpose != ai.ReceiptPage {
		return nil
	}
	input := []imageMessage{{Role: "user", Content: []imageContent{{Type: "input_text", Text: string(request.Input)}, {Type: "input_image", ImageURL: "data:image/png;base64," + base64.StdEncoding.EncodeToString(request.Image), Detail: "high"}}}}
	return []option.RequestOption{option.WithJSONSet("input", input)}
}
func (c *Client) validateOutput(data []byte, expected proposalInputCase, request ai.Request) error {
	if request.Purpose == ai.ReceiptPage {
		_, err := receiptcontract.Decode(data)
		return err
	}
	return validateProposal(data, expected)
}
