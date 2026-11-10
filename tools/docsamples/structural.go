package main

import "strings"

// Structural defects (DOC_*: a mis-fenced AILANG block, an unknown tag, a fence that
// swallows the blocks after it, an import that does not resolve) are never baselinable:
// they are not "this program fails ailang check", they are "the page does not show what
// its author wrote", and listing one hides every block it swallows.
//
// structuralExemptions is the ONLY way past one, and each entry is named here, with its
// reason and its exit. Keyed by "<source file> <block key>" — the SOURCE file, not the
// page key, so an exemption for a frozen prompt revision does not carry over to the next
// revision.
var structuralExemptions = map[string]string{
	"prompts/v0.16.12.md ddfa674b342f": "frozen prompt v0.16.12 (sha256-pinned by `ailang prompt freeze --check`; editing it needs a new " +
		"revision): the stray fence at line 569 swallows the ```ailang block at line 591, which is never checked. " +
		"Fix it in the next prompt revision (PR #2223) and delete this exemption.",
}

// exemptFor returns the exemption reason for a structural block, if it has one.
func exemptFor(b *Block) (string, bool) {
	why, ok := structuralExemptions[b.Source+" "+b.Key]
	return why, ok
}

// staleExemptions lists exemptions that no longer match a defect. One whose source file
// is still in scope is a failure (the defect was fixed: remove the entry). One whose
// source has left scope (the prompt moved to a new revision) is only a note, so a prompt
// revision is not blocked on editing this file.
func staleExemptions(blocks []*Block, outcomes map[*Block]*Outcome, sources map[string]bool) (fail, note []string) {
	matched := map[string]bool{}
	for _, b := range blocks {
		if o := outcomes[b]; o != nil && isStructural(o.Code) {
			matched[b.Source+" "+b.Key] = true
		}
	}
	for k := range structuralExemptions {
		if matched[k] {
			continue
		}
		if sources[strings.SplitN(k, " ", 2)[0]] {
			fail = append(fail, k)
		} else {
			note = append(note, k)
		}
	}
	return fail, note
}
