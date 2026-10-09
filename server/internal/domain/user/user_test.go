package user

import "testing"

func TestNormalizeEmail(t *testing.T) {
	ok := map[string]string{
		"  A@B.com ":         "a@b.com",
		"user@example.co.kr": "user@example.co.kr",
	}
	for in, want := range ok {
		got, err := NormalizeEmail(in)
		if err != nil || got != want {
			t.Errorf("NormalizeEmail(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"", "plain", "a@b", "@b.com", "a@", "Name <a@b.com>", "a b@c.com"} {
		if _, err := NormalizeEmail(in); err != ErrInvalidEmail {
			t.Errorf("NormalizeEmail(%q)는 ErrInvalidEmail이어야 한다: %v", in, err)
		}
	}
}

func TestValidateConsent(t *testing.T) {
	if err := ValidateConsent(true, true); err != nil {
		t.Errorf("둘 다 동의하면 통과해야 한다: %v", err)
	}
	for _, c := range [][2]bool{{false, true}, {true, false}, {false, false}} {
		if err := ValidateConsent(c[0], c[1]); err != ErrConsentRequired {
			t.Errorf("ValidateConsent(%v)는 필수 동의 오류여야 한다: %v", c, err)
		}
	}
}
