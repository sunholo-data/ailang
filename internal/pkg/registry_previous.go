package pkg

// PreviousPublishedVersion returns the most recent version of name in the
// index other than newVersion ("" if there is none). The index's Versions list
// includes the just-published version once a publish has landed, so it is
// skipped rather than assumed absent.
func PreviousPublishedVersion(index *RegistryIndex, name, newVersion string) string {
	if index == nil {
		return ""
	}
	for _, e := range index.Packages {
		if e.Name != name {
			continue
		}
		for i := len(e.Versions) - 1; i >= 0; i-- {
			if e.Versions[i] != newVersion {
				return e.Versions[i]
			}
		}
		return ""
	}
	return ""
}
