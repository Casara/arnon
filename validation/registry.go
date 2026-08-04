package validation

import "sync"

// Custom rules are schema-level metadata about a tag name, shared by
// every validator instance, the OpenAPI generator and the error
// mapper, which is why they live in a global registry rather than as
// per-instance state.
//
//nolint:gochecknoglobals // shared framework-wide tag metadata, see comment above
var (
	customRulesMu sync.RWMutex
	customRules   = make(map[string]CustomRule)
)

// RegisterCustomRule registers a custom validation tag globally.
//
// It is intended to be called once during application bootstrap, before
// any validator or OpenAPI document is built. Once registered, the tag
// is:
//
//   - applied automatically to every validator instance created via New
//     or Default
//   - used to map validation failures to problem details
//   - used to enrich generated OpenAPI schemas
//
// Registering a rule with a tag that was already registered replaces
// the previous definition.
func RegisterCustomRule(rule CustomRule) error {
	if rule.Tag == "" {
		return ErrCustomRuleTagEmpty
	}

	if rule.Func == nil {
		return ErrCustomRuleFuncNil
	}

	customRulesMu.Lock()
	defer customRulesMu.Unlock()

	customRules[rule.Tag] = rule

	return nil
}

// LookupCustomRule returns the custom rule registered for tag, if any.
func LookupCustomRule(tag string) (CustomRule, bool) {
	customRulesMu.RLock()
	defer customRulesMu.RUnlock()

	rule, ok := customRules[tag]

	return rule, ok
}

// registeredCustomRules returns a snapshot of all registered custom
// rules, used to seed newly created validator instances.
func registeredCustomRules() []CustomRule {
	customRulesMu.RLock()
	defer customRulesMu.RUnlock()

	rules := make([]CustomRule, 0, len(customRules))

	for _, rule := range customRules {
		rules = append(rules, rule)
	}

	return rules
}
