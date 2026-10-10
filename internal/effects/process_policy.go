package effects

// Child preserves immutable authorization/configuration with fresh process state.
// Copy policy maps so host changes cannot mutate another request's authority.
func (pc *ProcessContext) Child() *ProcessContext {
	c := NewProcessContext()
	c.Timeout, c.MaxOutput, c.HasAllowlist, c.Confined = pc.Timeout, pc.MaxOutput, pc.HasAllowlist, pc.Confined
	if pc.Allowlist != nil {
		c.Allowlist = make(map[string]string, len(pc.Allowlist))
		for k, v := range pc.Allowlist {
			c.Allowlist[k] = v
		}
	}
	if pc.Subcommands != nil {
		c.Subcommands = make(map[string][][]string, len(pc.Subcommands))
		for k, chains := range pc.Subcommands {
			for _, chain := range chains {
				c.Subcommands[k] = append(c.Subcommands[k], append([]string(nil), chain...))
			}
		}
	}
	return c
}
