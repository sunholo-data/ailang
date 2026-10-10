package claudegateway

import (
	"encoding/json"
	"log"

	"github.com/sunholo-data/ailang/internal/modelreg"
)

// UsageReceipt is a billing-only record emitted after durable gateway
// settlement. Usage contains only verified numeric categories; provider bodies,
// observations, headers, prompts and credentials cannot enter this schema.
type UsageReceipt struct {
	Event             string               `json:"event"`
	AccountID         string               `json:"account_id"`
	TaskID            string               `json:"task_id"`
	RequestID         string               `json:"request_id"`
	ProviderRequestID string               `json:"provider_request_id"`
	MessageID         string               `json:"message_id"`
	Model             string               `json:"model"`
	PricingRevision   string               `json:"pricing_revision"`
	CostMicroUSD      int64                `json:"cost_micro_usd"`
	Usage             modelreg.ClaudeUsage `json:"usage"`
}

func (g *Gateway) reportSettled(receipt UsageReceipt) {
	receipt.ProviderRequestID = diagnosticID(receipt.ProviderRequestID)
	receipt.MessageID = diagnosticID(receipt.MessageID)
	if g.Receipt != nil {
		g.Receipt(receipt)
		return
	}
	encoded, _ := json.Marshal(receipt)
	log.Printf("%s", encoded)
}
