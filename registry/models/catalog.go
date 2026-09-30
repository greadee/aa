package models

import v2 "github.com/greadee/aa/contracts/go/v2"

// DefaultCatalog returns the bundled model catalog as durable specs, mirrored
// from the inference service's advisory catalog
// (runtime/inference/src/aa_inference/catalog/data/catalog.toml). It is a
// snapshot; the inference catalog remains the runtime source and this list is
// refreshed when it changes.
func DefaultCatalog() []ModelSpec {
	return []ModelSpec{
		spec("ollama", "qwen3.5:9b", 131072, true, []string{"general", "coding", "tool_use", "structured_output", "agentic"}, nil),
		spec("ollama", "qwen3.5:4b", 131072, true, []string{"general", "coding", "tool_use"}, nil),
		spec("ollama", "qwen2.5-coder:7b", 32768, false, []string{"coding", "general"}, nil),
		spec("ollama", "llama3.1:8b", 131072, true, []string{"general", "tool_use"}, nil),
		spec("ollama", "mistral-nemo:12b", 131072, false, []string{"general", "reasoning"}, nil),
		spec("ollama", "qwen3.5:32b", 131072, true, []string{"general", "coding", "reasoning", "tool_use"}, nil),
		spec("ollama", "deepseek-coder-v2:16b", 131072, false, []string{"coding", "general"}, nil),
		spec("deepseek", "deepseek-chat", 131072, true, []string{"general", "coding", "tool_use", "structured_output", "agentic", "long_context"}, &ModelCost{InputPer1M: 0.14, OutputPer1M: 0.28}),
		spec("deepseek", "deepseek-reasoner", 131072, false, []string{"general", "coding", "reasoning", "structured_output", "long_context"}, &ModelCost{InputPer1M: 0.55, OutputPer1M: 2.19}),
		spec("openai-compatible", "gpt-4o-mini", 131072, true, []string{"general", "coding", "tool_use", "structured_output", "vision"}, &ModelCost{InputPer1M: 0.15, OutputPer1M: 0.60}),
	}
}

// ModelCost is the catalog's per-million-token cost, in USD.
type ModelCost struct {
	InputPer1M  float64
	OutputPer1M float64
}

func spec(provider, identifier string, contextWindow int, toolUse bool, capabilities []string, cost *ModelCost) ModelSpec {
	locality := "local"
	if cost != nil {
		locality = "cloud"
	}
	s := ModelSpec{
		Envelope:      v2.Envelope{ContractVersion: "2.0", Kind: "model_spec", ID: "mdl_" + provider + "_" + identifier},
		Provider:      provider,
		Identifier:    identifier,
		ModelVersion:  "catalog",
		Capabilities:  append([]string(nil), capabilities...),
		ContextWindow: contextWindow,
		ToolUse:       toolUse,
		Locality:      locality,
		Available:     true,
	}
	if cost != nil {
		in := cost.InputPer1M / 1000
		out := cost.OutputPer1M / 1000
		s.CostPer1KInput = &in
		s.CostPer1KOut = &out
	}
	return s
}
