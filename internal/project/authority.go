package project

func (s NestedConfigSettings) AuthorityFor(delegate NestedDelegate, subsystem AuthoritySubsystem) AuthorityPolicy {
	rule, ok := delegate.Authority[subsystem]
	if ok && rule.Default != "" {
		return rule.Default
	}
	return s.DefaultAuthority
}

func (s NestedConfigSettings) AuthorityForCheck(delegate NestedDelegate, subsystem AuthoritySubsystem, kind, family string) AuthorityPolicy {
	rule, ok := delegate.Authority[subsystem]
	if ok {
		if policy, found := rule.Kinds[kind]; found {
			return policy
		}
		if policy, found := rule.Families[family]; found {
			return policy
		}
		if rule.Default != "" {
			return rule.Default
		}
	}
	return s.DefaultAuthority
}

func AuthoritySubsystems() []AuthoritySubsystem {
	out := make([]AuthoritySubsystem, len(authoritySubsystems))
	copy(out, authoritySubsystems)
	return out
}
