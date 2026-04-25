// Package component provides reusable UI components for the TUI.
// This file contains tests for the animation system.
package component

import (
	"testing"
	"time"
)

func TestNewAnimation(t *testing.T) {
	anim := NewAnimation(AnimationFadeIn, 100*time.Millisecond)

	if anim.Type != AnimationFadeIn {
		t.Errorf("Type = %v, want AnimationFadeIn", anim.Type)
	}

	if anim.Progress != 0.0 {
		t.Errorf("Progress = %v, want 0.0", anim.Progress)
	}

	if anim.Duration != 100*time.Millisecond {
		t.Errorf("Duration = %v, want 100ms", anim.Duration)
	}

	if anim.Complete {
		t.Error("Complete should be false")
	}
}

func TestAnimation_Update(t *testing.T) {
	anim := NewAnimation(AnimationFadeIn, 50*time.Millisecond)

	// Initially not complete
	if anim.Complete {
		t.Error("Animation should not be complete initially")
	}

	// Wait for animation to complete
	time.Sleep(60 * time.Millisecond)
	cmd := anim.Update()

	// Should be complete now
	if !anim.Complete {
		t.Error("Animation should be complete after duration")
	}

	if anim.Progress != 1.0 {
		t.Errorf("Progress = %v, want 1.0", anim.Progress)
	}

	// Complete animation should return nil
	if cmd != nil {
		t.Error("Update should return nil when complete")
	}
}

func TestEaseOutCubic(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{0.0, 0.0},
		{1.0, 1.0},
	}

	for _, tt := range tests {
		result := EaseOutCubic(tt.input)
		if result != tt.expected {
			t.Errorf("EaseOutCubic(%v) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestEaseInOutCubic(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{0.0, 0.0},
		{1.0, 1.0},
	}

	for _, tt := range tests {
		result := EaseInOutCubic(tt.input)
		if result != tt.expected {
			t.Errorf("EaseInOutCubic(%v) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestEaseOutExpo(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{0.0, 0.0},
		{1.0, 1.0},
	}

	for _, tt := range tests {
		result := EaseOutExpo(tt.input)
		if result != tt.expected {
			t.Errorf("EaseOutExpo(%v) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestNewDialogAnimation(t *testing.T) {
	anim := NewDialogAnimation()

	if anim == nil {
		t.Fatal("NewDialogAnimation() returned nil")
	}

	if anim.Visible {
		t.Error("Visible should be false initially")
	}

	if anim.IsVisible() {
		t.Error("IsVisible() should return false initially")
	}
}

func TestDialogAnimation_Open(t *testing.T) {
	anim := NewDialogAnimation()

	cmd := anim.Open()

	if !anim.Visible {
		t.Error("Visible should be true after Open()")
	}

	if anim.Animation.Type != AnimationFadeIn {
		t.Errorf("Animation.Type = %v, want AnimationFadeIn", anim.Animation.Type)
	}

	if cmd == nil {
		t.Error("Open() should return a command")
	}
}

func TestDialogAnimation_Close(t *testing.T) {
	anim := NewDialogAnimation()
	anim.Open()

	cmd := anim.Close()

	if anim.Animation.Type != AnimationFadeOut {
		t.Errorf("Animation.Type = %v, want AnimationFadeOut", anim.Animation.Type)
	}

	if cmd == nil {
		t.Error("Close() should return a command")
	}
}

func TestDialogAnimation_GetOpacity(t *testing.T) {
	anim := NewDialogAnimation()

	// Not visible, no animation
	opacity := anim.GetOpacity()
	if opacity != 0.0 {
		t.Errorf("GetOpacity() = %v, want 0.0 when not visible", opacity)
	}

	// After open
	anim.Open()
	// Note: Without actual frame updates, Progress will be 0 but time has elapsed
	opacity = anim.GetOpacity()
	// At the very start, opacity should be very small (near 0)
	if opacity > 0.1 {
		t.Errorf("GetOpacity() = %v, want near 0.0 at start of animation", opacity)
	}
}

func TestDialogAnimation_GetOffset(t *testing.T) {
	anim := NewDialogAnimation()

	// No animation
	offset := anim.GetOffset()
	if offset != 0 {
		t.Errorf("GetOffset() = %v, want 0", offset)
	}

	// After open
	anim.Open()
	offset = anim.GetOffset()
	// At start of animation, offset should be max
	if offset == 0 {
		t.Error("GetOffset() should not be 0 at animation start")
	}
}

func TestNewToastAnimation(t *testing.T) {
	anim := NewToastAnimation(true, 2*time.Second)

	if anim == nil {
		t.Fatal("NewToastAnimation() returned nil")
	}

	if !anim.AutoDismiss {
		t.Error("AutoDismiss should be true")
	}

	if anim.DismissTimer == nil {
		t.Error("DismissTimer should not be nil")
	}
}

func TestToastAnimation_Show(t *testing.T) {
	anim := NewToastAnimation(false, 0)

	cmd := anim.Show()

	if !anim.Visible {
		t.Error("Visible should be true after Show()")
	}

	if anim.Animation.Type != AnimationSlideUp {
		t.Errorf("Animation.Type = %v, want AnimationSlideUp", anim.Animation.Type)
	}

	if cmd == nil {
		t.Error("Show() should return a command")
	}
}

func TestToastAnimation_Dismiss(t *testing.T) {
	anim := NewToastAnimation(false, 0)
	anim.Show()

	cmd := anim.Dismiss()

	if anim.Animation.Type != AnimationFadeOut {
		t.Errorf("Animation.Type = %v, want AnimationFadeOut", anim.Animation.Type)
	}

	if cmd == nil {
		t.Error("Dismiss() should return a command")
	}
}

func TestNewScrollAnimation(t *testing.T) {
	anim := NewScrollAnimation(0, 100)

	if anim == nil {
		t.Fatal("NewScrollAnimation() returned nil")
	}

	if anim.StartPos != 0 {
		t.Errorf("StartPos = %v, want 0", anim.StartPos)
	}

	if anim.TargetPos != 100 {
		t.Errorf("TargetPos = %v, want 100", anim.TargetPos)
	}

	if anim.Current != 0 {
		t.Errorf("Current = %v, want 0", anim.Current)
	}
}

func TestScrollAnimation_GetPosition(t *testing.T) {
	anim := NewScrollAnimation(0, 100)

	pos := anim.GetPosition()
	if pos != 0 {
		t.Errorf("GetPosition() = %v, want 0", pos)
	}
}

func TestScrollAnimation_IsComplete(t *testing.T) {
	anim := NewScrollAnimation(0, 100)

	if anim.IsComplete() {
		t.Error("IsComplete() should return false initially")
	}
}
