package remarks

import (
	"strings"
	"testing"
)

func TestIndex_IsStableAndKeyDerived(t *testing.T) {
	first := Index("sk-abc")
	if first != Index("sk-abc") {
		t.Fatal("Index() is not stable for the same key")
	}
	if first == Index("sk-abd") {
		t.Fatal("Index() collides for different keys")
	}
	if len(first) != 64 || strings.ToLower(first) != first {
		t.Fatalf("Index() = %q, want a 64 character lowercase hex digest", first)
	}
	if strings.Contains(first, "sk-abc") {
		t.Fatalf("Index() %q leaks the key material", first)
	}
}

func TestValidate_EnforcesRuneLimit(t *testing.T) {
	if err := Validate(strings.Repeat("备", MaxLength)); err != nil {
		t.Fatalf("Validate() at the limit error = %v, want nil", err)
	}
	if err := Validate(strings.Repeat("备", MaxLength+1)); err == nil {
		t.Fatal("Validate() over the limit returned no error")
	}
	if err := Validate(""); err != nil {
		t.Fatalf("Validate(\"\") error = %v, want nil (empty clears the remark)", err)
	}
}

func TestNormalize_TrimsWhitespace(t *testing.T) {
	if got := Normalize("  dev key \n"); got != "dev key" {
		t.Fatalf("Normalize() = %q, want %q", got, "dev key")
	}
}

func TestValidateKey_RejectsBlankKeys(t *testing.T) {
	if err := ValidateKey("sk-real"); err != nil {
		t.Fatalf("ValidateKey() error = %v, want nil", err)
	}
	if err := ValidateKey("   "); err != ErrEmptyKey {
		t.Fatalf("ValidateKey(\"   \") error = %v, want ErrEmptyKey", err)
	}
}
