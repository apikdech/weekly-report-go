package llm

import (
	anyllm "github.com/mozilla-ai/any-llm-go"
	"github.com/mozilla-ai/any-llm-go/providers/anthropic"
	"github.com/mozilla-ai/any-llm-go/providers/deepseek"
	"github.com/mozilla-ai/any-llm-go/providers/gemini"
	"github.com/mozilla-ai/any-llm-go/providers/groq"
	"github.com/mozilla-ai/any-llm-go/providers/ollama"
	"github.com/mozilla-ai/any-llm-go/providers/openai"
)

// providerConstructor matches how we build an any-llm-go Provider from options.
// Native packages return concrete *Provider types; wrappers convert to anyllm.Provider.
type providerConstructor func(opts ...anyllm.Option) (anyllm.Provider, error)

var providerRegistry = map[ProviderType]providerConstructor{
	Gemini:    func(o ...anyllm.Option) (anyllm.Provider, error) { return gemini.New(o...) },
	OpenAI:    func(o ...anyllm.Option) (anyllm.Provider, error) { return openai.New(o...) },
	Anthropic: func(o ...anyllm.Option) (anyllm.Provider, error) { return anthropic.New(o...) },
	Groq:      func(o ...anyllm.Option) (anyllm.Provider, error) { return groq.New(o...) },
	Ollama:    func(o ...anyllm.Option) (anyllm.Provider, error) { return ollama.New(o...) },
	DeepSeek:  func(o ...anyllm.Option) (anyllm.Provider, error) { return deepseek.New(o...) },
}
