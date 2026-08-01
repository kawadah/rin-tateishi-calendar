package config

import "testing"

func TestDefault(t *testing.T) {
	c := Default()
	if c.MemberNo != 231613 {
		t.Errorf("MemberNo = %d, want 231613", c.MemberNo)
	}
	if c.CanaryFrom.Year != 2024 || c.CanaryTo.Year != 2025 {
		t.Errorf("unexpected canary window %v..%v", c.CanaryFrom, c.CanaryTo)
	}
}

func TestFromEnvOverridesMember(t *testing.T) {
	t.Setenv("SOURCE_MEMBER_NO", "999")
	c, err := FromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if c.MemberNo != 999 {
		t.Errorf("MemberNo = %d, want 999", c.MemberNo)
	}
}

func TestFromEnvRejectsBadMember(t *testing.T) {
	t.Setenv("SOURCE_MEMBER_NO", "notanumber")
	if _, err := FromEnv(); err == nil {
		t.Fatal("expected error for non-numeric SOURCE_MEMBER_NO")
	}
}
