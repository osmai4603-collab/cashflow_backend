package sequence_test

import (
	"context"
	"testing"
	"time"

	"cashflow_backend/internal/domain/sequence"
	platformsequence "cashflow_backend/internal/platform/sequence"
)

func TestFormat(t *testing.T) {
	ts := time.Date(2024, 3, 5, 12, 0, 0, 0, time.UTC)

	s := &sequence.Sequence{
		Prefix:  "INV/%(year)s/",
		Suffix:  "/%(month)s%(day)s",
		Padding: 6,
	}
	got := platformsequence.Format(s, 42, ts)
	want := "INV/2024/000042/0305"
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestFormat_DefaultPadding(t *testing.T) {
	s := &sequence.Sequence{Padding: 0}
	got := platformsequence.Format(s, 7, time.Now())
	want := "00007"
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestExpandDateTokens(t *testing.T) {
	ts := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	got := platformsequence.ExpandDateTokens("Y%(year)s-M%(month)s-D%(day)s", ts)
	want := "Y2024-M12-D31"
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}

	if got := platformsequence.ExpandDateTokens("", ts); got != "" {
		t.Errorf("expected empty output for empty template, got %q", got)
	}
}

type fakeNumberSource struct {
	values []int
}

func (f *fakeNumberSource) NextValue(ctx context.Context, id int64) (int, error) {
	if len(f.values) == 0 {
		return 0, nil
	}
	v := f.values[0]
	f.values = f.values[1:]
	return v, nil
}

func TestGenerator_Next(t *testing.T) {
	ctx := context.Background()
	gen := platformsequence.NewGenerator(&fakeNumberSource{values: []int{1, 2}})
	ts := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	ref, err := gen.Next(ctx, &sequence.Sequence{Prefix: "SO/", Padding: 4}, ts)
	if err != nil {
		t.Fatalf("unexpected error getting next: %v", err)
	}
	if ref != "SO/0001" {
		t.Errorf("expected SO/0001, got %q", ref)
	}

	ref, err = gen.Next(ctx, &sequence.Sequence{Prefix: "SO/", Padding: 4}, ts)
	if err != nil {
		t.Fatalf("unexpected error getting next: %v", err)
	}
	if ref != "SO/0002" {
		t.Errorf("expected SO/0002, got %q", ref)
	}
}

func TestGenerator_NoSource(t *testing.T) {
	gen := platformsequence.NewGenerator(nil)
	_, err := gen.Next(context.Background(), &sequence.Sequence{}, time.Now())
	if err == nil {
		t.Fatalf("expected error for nil number source, got nil")
	}
}
