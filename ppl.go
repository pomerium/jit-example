package main

import (
	"time"
)

type JITUser struct {
	Email   string
	Expires time.Time
}

func toPPL(jitUsers []JITUser) Policy_Ppl {
	rules := make([]PPLRule, 0)

	for _, jitUser := range jitUsers {
		criteria := []PPLCriteria{
			{"email": jitUser.Email},
		}
		if jitUser.Expires.IsZero() {
			criteria = append(criteria, map[string]any{
				"reject": true,
			})
		} else {
			criteria = append(criteria, map[string]any{
				"date": map[string]any{
					"before": jitUser.Expires.Format(time.RFC3339),
				},
			})
		}
		rules = append(rules, PPLRule{
			Allow: &PPLRuleBody{
				And: &criteria,
			},
		})
	}
	var ppl Policy_Ppl
	_ = ppl.FromPolicyPpl1(rules)
	return ppl
}

func fromPPL(ppl Policy_Ppl) []JITUser {
	var rules []PPLRule
	if r, err := ppl.AsPPLRule(); err == nil {
		rules = append(rules, r)
	} else if rs, err := ppl.AsPolicyPpl1(); err == nil {
		rules = append(rules, rs...)
	}

	var jitUsers []JITUser
	for _, r := range rules {
		if r.Allow == nil || r.Allow.And == nil {
			continue
		}

		var jitUser JITUser
		for _, c := range *r.Allow.And {
			if email, ok := c["email"].(string); ok {
				jitUser.Email = email
			}
			if m, ok := c["date"].(map[string]any); ok {
				if tm, err := time.Parse(time.RFC3339, m["before"].(string)); err == nil {
					jitUser.Expires = tm
				}
			}
		}
		if jitUser.Email != "" {
			jitUsers = append(jitUsers, jitUser)
		}
	}

	return jitUsers
}
