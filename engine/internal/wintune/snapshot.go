//go:build windows

package wintune

import (
	"errors"
	"fmt"
	"time"
)

// SnapshotSchemaVersion is the persisted snapshot format version. Bump it whenever
// the captured value set changes so an older snapshot is never silently misread.
const SnapshotSchemaVersion = 1

// RegKind distinguishes the registry value types Remotune round-trips. Preserving the
// type matters: several affected values are REG_SZ holding digits, and rewriting them
// as REG_DWORD changes their meaning.
type RegKind string

const (
	RegKindDWord  RegKind = "dword"
	RegKindString RegKind = "string"
)

// RegValue is a single registry value captured with enough fidelity to be written back
// exactly as it was found.
type RegValue struct {
	Kind  RegKind `json:"kind"`
	DWord uint32  `json:"dword,omitempty"`
	Str   string  `json:"str,omitempty"`
}

func (v RegValue) String() string {
	if v.Kind == RegKindString {
		return v.Str
	}
	return fmt.Sprintf("%d", v.DWord)
}

// Equal reports whether two captured registry values are identical.
func (v RegValue) Equal(o RegValue) bool {
	if v.Kind != o.Kind {
		return false
	}
	if v.Kind == RegKindString {
		return v.Str == o.Str
	}
	return v.DWord == o.DWord
}

// VisualEffectsSnapshot captures all three layers that together define the user's
// Visual Effects state. No single layer is sufficient:
//
//	SPI      per-effect values readable through SystemParametersInfo
//	Registry values that have no SPI accessor, plus FontSmoothing's real type
//	Mask     UserPreferencesMask, opaque, preserved verbatim
type VisualEffectsSnapshot struct {
	SPI      map[string]int32    `json:"spi"`
	Registry map[string]RegValue `json:"registry"`
	Mask     []byte              `json:"userPreferencesMask"`
}

// TaskbarSnapshot captures auto-hide from both the live and the persisted layer.
//
// Only the desired auto-hide value is restored, never the whole StuckRects3 blob,
// so unrelated taskbar settings the user changes meanwhile are not reverted.
type TaskbarSnapshot struct {
	AutoHide            bool   `json:"autoHide"`
	ABMState            uint32 `json:"abmState"`
	LivePersistedAgreed bool   `json:"livePersistedAgreed"`
}

// Snapshot is the durable recovery record written before any mutation.
type Snapshot struct {
	SchemaVersion int                    `json:"schemaVersion"`
	CapturedAt    time.Time              `json:"capturedAt"`
	Machine       string                 `json:"machine"`
	OSBuild       string                 `json:"osBuild"`
	VisualEffects *VisualEffectsSnapshot `json:"visualEffects,omitempty"`
	Taskbar       *TaskbarSnapshot       `json:"taskbar,omitempty"`
}

// ErrUnsupportedSchema is returned when a snapshot cannot be trusted.
var ErrUnsupportedSchema = errors.New("unsupported snapshot schema version")

// Validate rejects snapshots that must not be used for restoration. A snapshot that
// fails validation is never guessed around: the caller reports that no restorable
// state exists rather than inventing one.
func (s *Snapshot) Validate() error {
	if s == nil {
		return errors.New("snapshot is nil")
	}
	if s.SchemaVersion != SnapshotSchemaVersion {
		return fmt.Errorf("%w: got %d, want %d", ErrUnsupportedSchema, s.SchemaVersion, SnapshotSchemaVersion)
	}
	if s.VisualEffects == nil && s.Taskbar == nil {
		return errors.New("snapshot records no categories")
	}
	if ve := s.VisualEffects; ve != nil {
		if len(ve.Mask) == 0 {
			return errors.New("visual effects snapshot has no UserPreferencesMask")
		}
		if len(ve.SPI) == 0 {
			return errors.New("visual effects snapshot has no SPI values")
		}
		if len(ve.Registry) == 0 {
			return errors.New("visual effects snapshot has no registry values")
		}
	}
	return nil
}

// Category names a unit of automation that can succeed or fail independently.
type Category string

const (
	CategoryVisualEffects Category = "visualEffects"
	CategoryTaskbar       Category = "taskbar"
)

// CategoryResult records the outcome of one category's apply or restore.
//
// Verified is only true when the observable outcome was re-read and matched. A write
// that returned success without a confirming read is not reported as verified.
type CategoryResult struct {
	Category Category
	Changed  bool
	Verified bool
	Err      error
}

// Result aggregates per-category outcomes for one transition.
type Result struct {
	Categories []CategoryResult
}

// Failed reports whether any category errored.
func (r Result) Failed() bool {
	for _, c := range r.Categories {
		if c.Err != nil {
			return true
		}
	}
	return false
}

// Partial reports whether some categories succeeded while others failed. A partial
// transition must never be presented as complete success, and its recovery data must
// be retained.
func (r Result) Partial() bool {
	var ok, bad int
	for _, c := range r.Categories {
		if c.Err != nil {
			bad++
		} else {
			ok++
		}
	}
	return ok > 0 && bad > 0
}

// FullyVerified reports whether every category succeeded and was confirmed by re-read.
func (r Result) FullyVerified() bool {
	if len(r.Categories) == 0 {
		return false
	}
	for _, c := range r.Categories {
		if c.Err != nil || !c.Verified {
			return false
		}
	}
	return true
}

// Err collapses the per-category errors into one, or nil.
func (r Result) Err() error {
	var errs []error
	for _, c := range r.Categories {
		if c.Err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", c.Category, c.Err))
		}
	}
	return errors.Join(errs...)
}
