package inspectorui

import "testing"

func TestMaxMinInt(t *testing.T) {
	if MaxInt(2, 5) != 5 {
		t.Fatalf("MaxInt(2,5) should be 5")
	}
	if MaxInt(9, 1) != 9 {
		t.Fatalf("MaxInt(9,1) should be 9")
	}
	if MinInt(2, 5) != 2 {
		t.Fatalf("MinInt(2,5) should be 2")
	}
	if MinInt(9, 1) != 1 {
		t.Fatalf("MinInt(9,1) should be 1")
	}
}

func TestFormatStatus(t *testing.T) {
	got := FormatStatus("a", "", "b", "", "c")
	want := "a │ b │ c"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestPadRight(t *testing.T) {
	if got := PadRight("abc", 5); got != "abc  " {
		t.Fatalf("got %q, want %q", got, "abc  ")
	}
	if got := PadRight("abcdef", 4); got != "abcd" {
		t.Fatalf("got %q, want %q", got, "abcd")
	}
}
