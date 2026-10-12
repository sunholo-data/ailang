package configdriven

import "github.com/sunholo-data/ailang/internal/config"

// WithFrozenAuthEnvironment captures only this spec's referenced environment
// values, without validation, file reads or network I/O. Missing credentials
// still fail when an authorized AI operation asks for them. Eager providers
// omitting this option retain per-call environment refresh.
func WithFrozenAuthEnvironment() Option {
	return func(p *Provider) {
		captured := make(map[string]string)
		if name := p.spec.Auth.Env; name != "" {
			captured[name] = config.Raw(name)
		}
		for _, header := range p.spec.AuthHeaders {
			for _, match := range envInterpolationPattern.FindAllStringSubmatch(header, -1) {
				captured[match[1]] = config.Raw(match[1])
			}
		}
		p.envLookup = func(name string) string { return captured[name] }
	}
}
