package openai

import (
	_ "embed"
	"encoding/json"
	"errors"

	ai "github.com/pchkauu/want-keep/backend/internal/ai/domain"
)

//go:embed runtime_contract.json
var runtimeContractJSON []byte

type route struct {
	MaximumInputTokens  int64 `json:"maximum_input_tokens"`
	MaximumOutputTokens int64 `json:"maximum_output_tokens"`
}

type runtimeContract struct {
	Kind                    string               `json:"kind"`
	SourceVersion           string               `json:"source_version"`
	SourceFingerprint       string               `json:"source_fingerprint"`
	RuntimeSchemaAdaptation string               `json:"runtime_schema_adaptation"`
	PromptFingerprint       string               `json:"prompt_fingerprint"`
	SchemaFingerprint       string               `json:"schema_fingerprint"`
	ConfigFingerprint       string               `json:"config_fingerprint"`
	Prompt                  string               `json:"prompt"`
	Schema                  map[string]any       `json:"schema"`
	Model                   string               `json:"model"`
	Qualification           string               `json:"qualification"`
	ReasoningEffort         string               `json:"reasoning_effort"`
	ReasoningMode           string               `json:"reasoning_mode"`
	ServiceTier             string               `json:"service_tier"`
	Store                   bool                 `json:"store"`
	Background              bool                 `json:"background"`
	PromptCacheMode         string               `json:"prompt_cache_mode"`
	Truncation              string               `json:"truncation"`
	ParallelToolCalls       bool                 `json:"parallel_tool_calls"`
	GlobalMaximumInput      int64                `json:"global_maximum_input_tokens"`
	MonthlyLimit            string               `json:"monthly_limit_usd"`
	MaximumFamilyConcurrent int                  `json:"maximum_family_concurrency"`
	ProductionAdmitted      bool                 `json:"production_admitted"`
	Routes                  map[ai.Purpose]route `json:"routes"`
}

func loadRuntimeContract() (runtimeContract, error) {
	var contract runtimeContract
	if err := json.Unmarshal(runtimeContractJSON, &contract); err != nil {
		return contract, err
	}
	metadata := contract.metadata()
	if contract.Kind != "want_keep_openai_runtime_contract_v1" || contract.SourceVersion != string(ai.TerraXHigh) || contract.SourceFingerprint != "837e23ea8215423dc003129da9c99e4ea02ec37ad260199a53fe154cc8de222b" || contract.RuntimeSchemaAdaptation != "transaction_review_commands_v1" || contract.Prompt == "" || len(contract.Schema) == 0 || contract.ReasoningEffort != "xhigh" || contract.ReasoningMode != "standard" || contract.ServiceTier != "default" || contract.Store || contract.Background || contract.PromptCacheMode != "explicit" || contract.Truncation != "disabled" || contract.ParallelToolCalls || contract.GlobalMaximumInput != 262144 || contract.MonthlyLimit != "50" || contract.MaximumFamilyConcurrent != 2 || contract.ProductionAdmitted {
		return contract, errors.New("invalid OpenAI runtime contract")
	}
	if err := metadata.Validate(); err != nil {
		return contract, err
	}
	requiredRoutes := []ai.Purpose{ai.TransactionReview, ai.ReceiptPage, ai.ChatInsight, ai.ComplexClarification}
	if len(contract.Routes) != len(requiredRoutes) {
		return contract, errors.New("invalid OpenAI route limits")
	}
	for _, purpose := range requiredRoutes {
		limits, ok := contract.Routes[purpose]
		if !ok {
			return contract, errors.New("invalid OpenAI route limits")
		}
		maximumInput, maximumOutput := purpose.Limits()
		if limits.MaximumInputTokens != maximumInput || limits.MaximumOutputTokens != maximumOutput {
			return contract, errors.New("invalid OpenAI route limits")
		}
	}
	return contract, nil
}

func (c runtimeContract) metadata() ai.RuntimeContract {
	return ai.RuntimeContract{
		Model: ai.Model(c.Model), Qualification: ai.Qualification(c.Qualification),
		PromptFingerprint: c.PromptFingerprint, SchemaFingerprint: c.SchemaFingerprint,
		ConfigFingerprint: c.ConfigFingerprint,
	}
}
