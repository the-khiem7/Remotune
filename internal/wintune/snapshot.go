//go:build windows

package wintune

import (
	"errors"
	"fmt"
	"time"
)

const SnapshotSchemaVersion = 1

type RegKind string

const (
	RegKindDWord  RegKind = "dword"
	RegKindString RegKind = "string"
)

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
func (v RegValue) Equal(o RegValue) bool {
	if v.Kind != o.Kind {
		return false
	}
	if v.Kind == RegKindString {
		return v.Str == o.Str
	}
	return v.DWord == o.DWord
}

type VisualEffectsSnapshot struct {
	SPI      map[string]int32    `json:"spi"`
	Registry map[string]RegValue `json:"registry"`
	Mask     []byte              `json:"userPreferencesMask"`
}
type TaskbarSnapshot struct {
	AutoHide            bool   `json:"autoHide"`
	ABMState            uint32 `json:"abmState"`
	LivePersistedAgreed bool   `json:"livePersistedAgreed"`
}
type Snapshot struct {
	SchemaVersion int                    `json:"schemaVersion"`
	CapturedAt    time.Time              `json:"capturedAt"`
	Machine       string                 `json:"machine"`
	OSBuild       string                 `json:"osBuild"`
	VisualEffects *VisualEffectsSnapshot `json:"visualEffects,omitempty"`
	Taskbar       *TaskbarSnapshot       `json:"taskbar,omitempty"`
}

var ErrUnsupportedSchema = errors.New("unsupported snapshot schema version")

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

type Category string

const (
	CategoryVisualEffects Category = "visualEffects"
	CategoryTaskbar       Category = "taskbar"
)

type CategoryResult struct {
	Category Category
	Changed  bool
	Verified bool
	Err      error
}
type Result struct {
	Categories []CategoryResult
}

func (r Result) Failed() bool {
	for _, c := range r.Categories {
		if c.Err != nil {
			return true
		}
	}
	return false
}
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
func (r Result) Err() error {
	var errs []error
	for _, c := range r.Categories {
		if c.Err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", c.Category, c.Err))
		}
	}
	return errors.Join(errs...)
}
