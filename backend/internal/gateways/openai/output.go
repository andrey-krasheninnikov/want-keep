package openai

import (
	"encoding/json"
	"errors"
	ai "github.com/pchkauu/want-keep/backend/internal/ai/domain"
)

type proposalInputCase struct {
	ID     string `json:"id"`
	Source string `json:"source"`
}

func proposalExpectation(data []byte) (proposalInputCase, error) {
	var cases []proposalInputCase
	if err := json.Unmarshal(data, &cases); err != nil || len(cases) != 1 || cases[0].ID == "" || cases[0].Source == "" {
		return proposalInputCase{}, errors.New("invalid proposal input")
	}
	return cases[0], nil
}

func validateProposal(data []byte, expected proposalInputCase) error {
	if expected.ID != "case-1" || expected.Source != "ledger_revision" {
		return errors.New("invalid proposal")
	}
	_, err := ai.DecodeReviewOutput(data)
	return err
}

func hasRefusal(raw string) bool {
	var response struct {
		Output []struct {
			Content []struct {
				Type string `json:"type"`
			} `json:"content"`
		} `json:"output"`
	}
	if json.Unmarshal([]byte(raw), &response) != nil {
		return false
	}
	for _, output := range response.Output {
		for _, content := range output.Content {
			if content.Type == "refusal" {
				return true
			}
		}
	}
	return false
}
