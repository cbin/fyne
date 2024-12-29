package fyne

import "time"

// AnimationCurve represents an animation algorithm for calculating the progress through a timeline.
// Custom animations can be provided by implementing the "func(float32) float32" definition.
// The input parameter will start at 0.0 when an animation starts and travel up to 1.0 at which point it will end.
// A linear animation would return the same output value as is passed in.
type AnimationCurve func(float32) float32

// AnimationRepeatForever is an AnimationCount value that indicates it should not stop looping.
//
// Since: 2.0
const AnimationRepeatForever = -1

var (
	// AnimationEaseInOut is the default easing, it starts slowly, accelerates to the middle and slows to the end.
	//
	// Since: 2.0
	AnimationEaseInOut = animationEaseInOut
	// AnimationEaseIn starts slowly and accelerates to the end.
	//
	// Since: 2.0
	AnimationEaseIn = animationEaseIn
	// AnimationEaseOut starts at speed and slows to the end.
	//
	// Since: 2.0
	AnimationEaseOut = animationEaseOut
	// AnimationLinear is a linear mapping for animations that progress uniformly through their duration.
	//
	// Since: 2.0
	AnimationLinear = animationLinear
)

// Animation represents an animated element within a Fyne canvas.
// These animations may control individual objects or entire scenes.
//
// Since: 2.0
type Animation struct {
	AutoReverse bool
	Curve       AnimationCurve
	Duration    time.Duration
	RepeatCount int
	Tick        func(float32)
}

// IndefiniteAnimation represents an animation that continues indefinitely. It has no duration
// or curve, and when started, the Tick function will be called every frame until Stop is invoked.
//
// Since: 2.6
type IndefiniteAnimation struct {
	Tick func()

	isSetup   bool
	animation Animation
}

// NewAnimation creates a very basic animation where the callback function will be called for every
// rendered frame between [time.Now] and the specified duration. The callback values start at 0.0 and
// will be 1.0 when the animation completes.
//
// Since: 2.0
func NewAnimation(d time.Duration, fn func(float32)) *Animation {
	return &Animation{Duration: d, Tick: fn}
}

// NewIndefiniteAnimation creates an indefinite animation where the callback function will be called
// for every rendered frame once started, until stopped.
//
// Since: 2.6
func NewIndefiniteAnimation(fn func()) *IndefiniteAnimation {
	return &IndefiniteAnimation{Tick: fn}
}

// Start registers the animation with the application run-loop and starts its execution.
func (a *Animation) Start() {
	CurrentApp().Driver().StartAnimation(a)
}

// Start registers the animation with the application run-loop and starts its execution.
func (i *IndefiniteAnimation) Start() {
	i.setupAnimation()
	i.animation.Start()
}

// Stop will end this animation and remove it from the run-loop.
func (a *Animation) Stop() {
	CurrentApp().Driver().StopAnimation(a)
}

// Stop will end this animation and remove it from the run-loop.
func (i *IndefiniteAnimation) Stop() {
	i.setupAnimation()
	i.animation.Stop()
}

func (i *IndefiniteAnimation) setupAnimation() {
	if i.isSetup {
		return
	}

	i.animation = Animation{
		Tick: func(_ float32) {
			i.Tick()
		},
		Curve:       AnimationLinear, // any curve will work
		Duration:    1 * time.Second, // anything positive here will work
		RepeatCount: AnimationRepeatForever,
	}
	i.isSetup = true
}

func animationEaseIn(val float32) float32 {
	return val * val
}

func animationEaseInOut(val float32) float32 {
	if val <= 0.5 {
		return val * val * 2
	}

	return -1 + (4-val*2)*val
}

func animationEaseOut(val float32) float32 {
	return val * (2 - val)
}

func animationLinear(val float32) float32 {
	return val
}
