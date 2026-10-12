package effects

import "github.com/sunholo-data/ailang/internal/ai"

// ProviderLookup resolves a named provider without initializing a selected
// model handler. CLI preparation supplies an immutable snapshot.
type ProviderLookup func(string) (ai.Provider, bool)

// WithProviderLookup attaches an immutable, per-execution named-provider
// lookup. Hosts must install it before executing effects. It is independent
// of the selected handler and never triggers lazy handler initialization.
func (c *EffContext) WithProviderLookup(lookup ProviderLookup) *EffContext {
	c.providerLookup = lookup
	return c
}

// LookupProvider uses the execution's frozen lookup, or the existing global
// registry for eager hosts which have not installed a per-run snapshot.
func (c *EffContext) LookupProvider(name string) (ai.Provider, bool) {
	if c != nil && c.providerLookup != nil {
		return c.providerLookup(name)
	}
	return ai.GlobalProviderRegistry.Lookup(name)
}
