package main

// M-COORDINATOR-EXECUTION-TRUST M4, part 3 — say WHICH PLANE you are looking at.
//
// `messages health` already prints its store and project, which is necessary and
// not sufficient: `ailang-multivac` and `ailang-multivac-dev` differ by a suffix
// and are entirely separate planes, each with its own Firestore, topic prefix,
// coordinator and executor jobs. Reading one and concluding about the other is a
// mistake that is easy to make and expensive to unmake — the automatic image
// builds deploy to dev, while the plane most people mean by "the coordinator"
// is prod.
//
// So name the environment in words, not just the project id.

// planeLabel returns a short human name for a message-store project: "prod",
// "dev", "test", or "" when the project is not one of the known planes (in which
// case the caller should print nothing rather than guess).
func planeLabel(project string) string {
	switch project {
	case "ailang-multivac":
		return "prod"
	case "ailang-multivac-dev":
		return "dev"
	case "ailang-multivac-test":
		return "test"
	}
	return ""
}
