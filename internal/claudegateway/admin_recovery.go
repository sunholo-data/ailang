package claudegateway

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"unicode"

	"github.com/sunholo-data/ailang/internal/creditbudget"
)

func validRecoveryTask(id string) bool {
	return id != "" && len(id) <= 256 && id != "." && id != ".." && !strings.Contains(id, "/") && strings.TrimSpace(id) == id && strings.IndexFunc(id, unicode.IsControl) < 0
}

// Called only after admin authentication, the operator allowlist and account
// route checks. The domain authority owns all atomic recovery preconditions.
func (g *Gateway) recoveryAdmin(w http.ResponseWriter, r *http.Request, identity, action string) bool {
	if action != "requests" && action != "conservative-debit" && action != "external-debit" {
		return false
	}
	var result any
	var err error
	if action == "requests" && r.Method == http.MethodGet {
		query, queryErr := url.ParseQuery(r.URL.RawQuery)
		task := query.Get("task_id")
		if queryErr != nil || len(query) != 1 || len(query["task_id"]) != 1 || !validRecoveryTask(task) {
			fail(w, 400, "requests requires exactly one valid task_id")
			return true
		}
		result, err = g.Authority.Requests(r.Context(), AccountID, task)
	} else if r.Method == http.MethodPost && action != "requests" {
		if r.URL.RawQuery != "" {
			fail(w, 400, "recovery mutations do not accept query parameters")
			return true
		}
		if action == "conservative-debit" {
			var d creditbudget.ConservativeDebit
			if err = decodeAdmin(w, r, &d); err == nil {
				if d.AccountID != AccountID || d.Operator != "" {
					err = fmt.Errorf("invalid account or caller-supplied operator")
				}
			}
			if err != nil {
				fail(w, 400, err.Error())
				return true
			}
			d.Operator = identity
			err = g.Authority.ConservativeDebit(r.Context(), d)
		} else {
			var d creditbudget.ExternalDebit
			if err = decodeAdmin(w, r, &d); err == nil {
				if d.AccountID != AccountID || d.Operator != "" {
					err = fmt.Errorf("invalid account or caller-supplied operator")
				}
			}
			if err != nil {
				fail(w, 400, err.Error())
				return true
			}
			d.Operator = identity
			err = g.Authority.ExternalDebit(r.Context(), d)
		}
		if err == nil {
			result, err = g.Authority.Status(r.Context(), AccountID)
		}
	} else {
		fail(w, 404, "unsupported recovery method")
		return true
	}
	if err != nil {
		fail(w, 409, fmt.Sprintf("credit operation refused: %v", err))
		return true
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
	return true
}
