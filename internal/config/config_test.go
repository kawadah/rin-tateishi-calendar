package config

import (
	"testing"

	"github.com/kawadah/rin-tateishi-calendar/internal/event"
)

func TestDefault(t *testing.T) {
	c := Default()
	if c.MemberNo != 231613 {
		t.Errorf("MemberNo = %d, want 231613", c.MemberNo)
	}
	if c.CanaryFrom.Year != 2024 || c.CanaryTo.Year != 2025 {
		t.Errorf("unexpected canary window %v..%v", c.CanaryFrom, c.CanaryTo)
	}
	if want := (event.Date{Year: 2001, Month: 7, Day: 10}); c.Birthday != want {
		t.Errorf("Birthday = %v, want %v", c.Birthday, want)
	}
	if c.BirthdayName != "立石凛" {
		t.Errorf("BirthdayName = %q, want 立石凛", c.BirthdayName)
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
