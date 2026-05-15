package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

type Expense struct {
	Date     string  `json:"date"`
	Vendor   string  `json:"vendor"`
	Currency string  `json:"currency"`
	Amount   float64 `json:"amount"`
	Summary  string  `json:"summary"`
}

const systemPrompt = `You analyse receipts (photos and PDFs of physical and digital receipts) and extract the fields needed for a single line on an expense claim form. Always invoke the record_expense tool with your structured answer — do not respond with prose.

Conventions:
- "date" is the date the expense was incurred (the transaction date on the receipt), in ISO 8601 YYYY-MM-DD format. For airline tickets, use the date of travel (first leg if a return). For hotels, use the check-in date.
- "vendor" is the trading name of the merchant (e.g. "KLM", "Hotel Bristol", "Costa Coffee"). Omit legal suffixes (Ltd, GmbH, B.V.) unless they are part of the common name.
- "currency" is a three-letter ISO 4217 code. If the receipt shows £, $, or € without an explicit code, infer GBP, USD, or EUR respectively.
- "amount" is the final total including taxes and fees, in the receipt's native currency. Always a numeric value (e.g. 142.30), never a string.
- "summary" is a concise 3–10 word description suitable as a line on an expense claim. Include the route for travel ("BRS-LIN return"), nights for hotels ("2 nights, business trip"), or item type for meals ("Client dinner, 3 attendees"). Neutral, factual, no punctuation needed.

If a field is ambiguous, give your best inference rather than refusing. Partial data is better than a failed extraction.`

var recordExpenseTool = anthropic.ToolParam{
	Name:        "record_expense",
	Description: anthropic.String("Record a single expense extracted from a receipt."),
	InputSchema: anthropic.ToolInputSchemaParam{
		Properties: map[string]any{
			"date":     map[string]any{"type": "string", "description": "Transaction date in ISO 8601 YYYY-MM-DD format."},
			"vendor":   map[string]any{"type": "string", "description": "Trading name of the merchant."},
			"currency": map[string]any{"type": "string", "description": "ISO 4217 currency code, e.g. GBP, EUR, USD."},
			"amount":   map[string]any{"type": "number", "description": "Total amount including taxes/fees, in the receipt's native currency."},
			"summary":  map[string]any{"type": "string", "description": "3–10 word description for the expense-claim line."},
		},
		Required: []string{"date", "vendor", "currency", "amount", "summary"},
	},
}

func extractExpense(ctx context.Context, client *anthropic.Client, receiptPath string) (Expense, error) {
	blocks, cleanup, err := buildContentBlocks(receiptPath)
	if err != nil {
		return Expense{}, err
	}
	if cleanup != nil {
		defer cleanup()
	}

	resp, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_6,
		MaxTokens: 1024,
		System: []anthropic.TextBlockParam{{
			Text:         systemPrompt,
			CacheControl: anthropic.NewCacheControlEphemeralParam(),
		}},
		Tools: []anthropic.ToolUnionParam{{OfTool: &recordExpenseTool}},
		ToolChoice: anthropic.ToolChoiceUnionParam{
			OfTool: &anthropic.ToolChoiceToolParam{Name: "record_expense"},
		},
		Thinking: anthropic.ThinkingConfigParamUnion{
			OfDisabled: &anthropic.ThinkingConfigDisabledParam{},
		},
		Messages: []anthropic.MessageParam{anthropic.NewUserMessage(blocks...)},
	})
	if err != nil {
		return Expense{}, err
	}

	for _, b := range resp.Content {
		if tu, ok := b.AsAny().(anthropic.ToolUseBlock); ok {
			var exp Expense
			if err := json.Unmarshal([]byte(tu.JSON.Input.Raw()), &exp); err != nil {
				return Expense{}, fmt.Errorf("parse tool input: %w", err)
			}
			return exp, nil
		}
	}
	return Expense{}, fmt.Errorf("no tool_use block in response (stop_reason=%s)", resp.StopReason)
}

func buildContentBlocks(receiptPath string) ([]anthropic.ContentBlockParamUnion, func(), error) {
	ext := strings.ToLower(filepath.Ext(receiptPath))
	prompt := anthropic.NewTextBlock("Extract the expense details from this receipt and call record_expense.")

	if ext == ".pdf" {
		b, err := os.ReadFile(receiptPath)
		if err != nil {
			return nil, nil, err
		}
		doc := anthropic.NewDocumentBlock(anthropic.Base64PDFSourceParam{
			Data: base64.StdEncoding.EncodeToString(b),
		})
		return []anthropic.ContentBlockParamUnion{doc, prompt}, nil, nil
	}

	var mediaType string
	var data []byte
	var cleanup func()

	switch ext {
	case ".jpg", ".jpeg":
		mediaType = "image/jpeg"
		b, err := os.ReadFile(receiptPath)
		if err != nil {
			return nil, nil, err
		}
		data = b
	case ".png":
		mediaType = "image/png"
		b, err := os.ReadFile(receiptPath)
		if err != nil {
			return nil, nil, err
		}
		data = b
	case ".heic", ".heif":
		tmp, err := os.CreateTemp("", "expense-sheet-*.jpg")
		if err != nil {
			return nil, nil, err
		}
		tmp.Close()
		cmd := exec.Command("sips",
			"-s", "format", "jpeg",
			"-s", "formatOptions", "80",
			"-Z", "2000",
			receiptPath, "--out", tmp.Name(),
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			os.Remove(tmp.Name())
			return nil, nil, fmt.Errorf("sips: %v: %s", err, strings.TrimSpace(string(out)))
		}
		b, err := os.ReadFile(tmp.Name())
		if err != nil {
			os.Remove(tmp.Name())
			return nil, nil, err
		}
		data = b
		cleanup = func() { os.Remove(tmp.Name()) }
		mediaType = "image/jpeg"
	default:
		return nil, nil, fmt.Errorf("unsupported extension: %s", ext)
	}

	img := anthropic.NewImageBlockBase64(mediaType, base64.StdEncoding.EncodeToString(data))
	return []anthropic.ContentBlockParamUnion{img, prompt}, cleanup, nil
}
