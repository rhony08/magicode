// Package component provides reusable UI components for the TUI.
// This file implements animation support for smooth transitions.
package component

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// AnimationType represents the type of animation
type AnimationType int

const (
	AnimationNone AnimationType = iota
	AnimationFadeIn
	AnimationFadeOut
	AnimationSlideUp
	AnimationSlideDown
)

// AnimationState tracks the current animation state
type AnimationState struct {
	// Current animation type
	Type AnimationType

	// Progress from 0.0 to 1.0
	Progress float64

	// Animation duration
	Duration time.Duration

	// Start time
	StartTime time.Time

	// Whether animation is complete
	Complete bool
}

// NewAnimation creates a new animation
func NewAnimation(animType AnimationType, duration time.Duration) AnimationState {
	return AnimationState{
		Type:      animType,
		Progress:  0.0,
		Duration:  duration,
		StartTime: time.Now(),
		Complete:  false,
	}
}

// Update updates the animation progress
func (a *AnimationState) Update() tea.Cmd {
	if a.Complete {
		return nil
	}

	elapsed := time.Since(a.StartTime)
	a.Progress = float64(elapsed) / float64(a.Duration)

	if a.Progress >= 1.0 {
		a.Progress = 1.0
		a.Complete = true
		return nil
	}

	// Request next frame (60fps target)
	return tea.Tick(time.Second/60, func(time.Time) tea.Msg {
		return AnimationTickMsg{}
	})
}

// AnimationTickMsg is sent on each animation frame
type AnimationTickMsg struct{}

// EaseOutCubic applies ease-out cubic easing function
func EaseOutCubic(t float64) float64 {
	return 1 - (1-t)*(1-t)*(1-t)
}

// EaseInOutCubic applies ease-in-out cubic easing function
func EaseInOutCubic(t float64) float64 {
	if t < 0.5 {
		return 4 * t * t * t
	}
	return 1 - (-2*t+2)*(-2*t+2)*(-2*t+2)/2
}

// EaseOutExpo applies ease-out exponential easing function
func EaseOutExpo(t float64) float64 {
	if t == 1 {
		return 1
	}
	return 1 - (1-t)*(1-t)*(1-t)*(1-t)
}

// DialogAnimation handles dialog open/close animations
type DialogAnimation struct {
	// Current animation
	Animation AnimationState

	// Target visibility
	Visible bool

	// Animation complete callback
	OnComplete func()
}

// NewDialogAnimation creates a new dialog animation
func NewDialogAnimation() *DialogAnimation {
	return &DialogAnimation{
		Animation:  NewAnimation(AnimationNone, 0),
		Visible:    false,
		OnComplete: nil,
	}
}

// Open starts the open animation
func (d *DialogAnimation) Open() tea.Cmd {
	d.Visible = true
	d.Animation = NewAnimation(AnimationFadeIn, 150*time.Millisecond)
	return d.Animation.Update()
}

// Close starts the close animation
func (d *DialogAnimation) Close() tea.Cmd {
	d.Animation = NewAnimation(AnimationFadeOut, 100*time.Millisecond)
	return d.Animation.Update()
}

// Update updates the animation
func (d *DialogAnimation) Update() tea.Cmd {
	cmd := d.Animation.Update()

	if d.Animation.Complete && d.OnComplete != nil {
		d.OnComplete()
	}

	return cmd
}

// GetOpacity returns the current opacity (0.0 to 1.0)
func (d *DialogAnimation) GetOpacity() float64 {
	if d.Animation.Type == AnimationNone {
		if d.Visible {
			return 1.0
		}
		return 0.0
	}

	progress := EaseOutCubic(d.Animation.Progress)

	switch d.Animation.Type {
	case AnimationFadeIn:
		return progress
	case AnimationFadeOut:
		return 1.0 - progress
	default:
		return 1.0
	}
}

// GetOffset returns the current vertical offset for slide animations
func (d *DialogAnimation) GetOffset() int {
	if d.Animation.Type == AnimationNone {
		return 0
	}

	progress := EaseOutExpo(d.Animation.Progress)
	maxOffset := 3 // Lines to slide

	switch d.Animation.Type {
	case AnimationFadeIn, AnimationSlideUp:
		return int(float64(maxOffset) * (1.0 - progress))
	case AnimationFadeOut, AnimationSlideDown:
		return int(float64(maxOffset) * progress)
	default:
		return 0
	}
}

// IsComplete returns true if animation is complete
func (d *DialogAnimation) IsComplete() bool {
	return d.Animation.Complete
}

// IsVisible returns true if dialog should be visible
func (d *DialogAnimation) IsVisible() bool {
	if d.Animation.Type == AnimationFadeOut && d.Animation.Complete {
		return false
	}
	return d.Visible
}

// ToastAnimation handles toast notification animations
type ToastAnimation struct {
	Animation    AnimationState
	Visible      bool
	AutoDismiss  bool
	DismissTimer *time.Timer
}

// NewToastAnimation creates a new toast animation
func NewToastAnimation(autoDismiss bool, dismissDelay time.Duration) *ToastAnimation {
	t := &ToastAnimation{
		Animation:   NewAnimation(AnimationNone, 0),
		Visible:     false,
		AutoDismiss: autoDismiss,
	}

	if autoDismiss {
		t.DismissTimer = time.NewTimer(dismissDelay)
	}

	return t
}

// Show starts the show animation
func (t *ToastAnimation) Show() tea.Cmd {
	t.Visible = true
	t.Animation = NewAnimation(AnimationSlideUp, 200*time.Millisecond)
	return t.Animation.Update()
}

// Dismiss starts the dismiss animation
func (t *ToastAnimation) Dismiss() tea.Cmd {
	t.Animation = NewAnimation(AnimationFadeOut, 150*time.Millisecond)
	return t.Animation.Update()
}

// Update updates the animation
func (t *ToastAnimation) Update() tea.Cmd {
	// Check auto-dismiss timer
	if t.AutoDismiss && t.DismissTimer != nil {
		select {
		case <-t.DismissTimer.C:
			if t.Visible && t.Animation.Type != AnimationFadeOut {
				return t.Dismiss()
			}
		default:
		}
	}

	return t.Animation.Update()
}

// GetOpacity returns the current opacity
func (t *ToastAnimation) GetOpacity() float64 {
	if t.Animation.Type == AnimationNone {
		if t.Visible {
			return 1.0
		}
		return 0.0
	}

	progress := EaseOutCubic(t.Animation.Progress)

	switch t.Animation.Type {
	case AnimationSlideUp:
		return progress
	case AnimationFadeOut:
		return 1.0 - progress
	default:
		return 1.0
	}
}

// GetOffset returns the vertical offset
func (t *ToastAnimation) GetOffset() int {
	if t.Animation.Type == AnimationNone {
		return 0
	}

	progress := EaseOutExpo(t.Animation.Progress)
	maxOffset := 2

	switch t.Animation.Type {
	case AnimationSlideUp:
		return int(float64(maxOffset) * (1.0 - progress))
	case AnimationFadeOut:
		return int(float64(maxOffset) * progress)
	default:
		return 0
	}
}

// IsComplete returns true if animation is complete
func (t *ToastAnimation) IsComplete() bool {
	return t.Animation.Complete
}

// IsVisible returns true if toast should be visible
func (t *ToastAnimation) IsVisible() bool {
	if t.Animation.Type == AnimationFadeOut && t.Animation.Complete {
		return false
	}
	return t.Visible
}

// ScrollAnimation handles smooth scroll animations
type ScrollAnimation struct {
	Animation AnimationState
	StartPos  int
	TargetPos int
	Current   int
}

// NewScrollAnimation creates a new scroll animation
func NewScrollAnimation(start, target int) *ScrollAnimation {
	distance := target - start
	if distance < 0 {
		distance = -distance
	}

	// Duration based on distance (max 300ms)
	duration := time.Duration(float64(distance) * 2)
	if duration > 300*time.Millisecond {
		duration = 300 * time.Millisecond
	}
	if duration < 100*time.Millisecond {
		duration = 100 * time.Millisecond
	}

	return &ScrollAnimation{
		Animation: NewAnimation(AnimationNone, duration),
		StartPos:  start,
		TargetPos: target,
		Current:   start,
	}
}

// Start starts the scroll animation
func (s *ScrollAnimation) Start() tea.Cmd {
	s.Animation = NewAnimation(AnimationNone, s.Animation.Duration)
	return s.Update()
}

// Update updates the animation
func (s *ScrollAnimation) Update() tea.Cmd {
	if s.Animation.Complete {
		s.Current = s.TargetPos
		return nil
	}

	cmd := s.Animation.Update()

	// Calculate current position with easing
	progress := EaseInOutCubic(s.Animation.Progress)
	s.Current = s.StartPos + int(float64(s.TargetPos-s.StartPos)*progress)

	return cmd
}

// GetPosition returns the current scroll position
func (s *ScrollAnimation) GetPosition() int {
	return s.Current
}

// IsComplete returns true if animation is complete
func (s *ScrollAnimation) IsComplete() bool {
	return s.Animation.Complete
}
