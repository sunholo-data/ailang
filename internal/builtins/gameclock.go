package builtins

import (
	"sync"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
)

// GameClock provides frame timing for game loops.
// All values are deterministic - set by the game engine, not real time.
var (
	gameClock   GameClockState
	gameClockMu sync.Mutex
)

// GameClockState holds the current frame timing state.
// Set by the game engine via SetGameClock before each frame.
type GameClockState struct {
	DeltaTime  float64 // Time since last frame in seconds (e.g., 0.016 for 60fps)
	TotalTime  float64 // Total game time in seconds since start
	FrameCount int     // Current frame number (starts at 0)
}

func init() {
	registerDeltaTime()
	registerTotalTime()
	registerFrameCount()
}

// SetGameClock sets the frame timing state.
// Call this at the start of each frame from the game loop.
func SetGameClock(deltaTime, totalTime float64, frameCount int) {
	gameClockMu.Lock()
	defer gameClockMu.Unlock()
	gameClock = GameClockState{
		DeltaTime:  deltaTime,
		TotalTime:  totalTime,
		FrameCount: frameCount,
	}
}

// GetGameClock returns the current frame timing state.
func GetGameClock() GameClockState {
	gameClockMu.Lock()
	defer gameClockMu.Unlock()
	return gameClock
}

// _game_delta_time: Get time since last frame in seconds
func registerDeltaTime() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/game",
		Name:    "_game_delta_time",
		NumArgs: 1, // Unit argument
		Effect:  "Clock",
		Type:    makeDeltaTimeType,
		Impl:    deltaTimeImpl,
		Metadata: &BuiltinMetadata{
			Description: "Get time since last frame in seconds",
			Params:      []ParamDoc{},
			Returns:     "Float representing seconds since last frame (e.g., 0.016 for 60fps)",
			Examples: []Example{
				{Code: "_game_delta_time()", Description: "Get delta time for physics calculations"},
				{Code: "let speed = 5.0 in position + velocity * _game_delta_time()", Description: "Frame-rate independent movement"},
			},
			Since:     "v0.5.1",
			Stability: StabilityStable,
			Tags:      []string{"game", "clock", "time", "frame"},
			Category:  "game",
		},
	})
	if err != nil {
		panic("failed to register _game_delta_time builtin: " + err.Error())
	}
}

func makeDeltaTimeType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.Unit()).Returns(T.Float()).Effects("Clock")
}

func deltaTimeImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	if err := ctx.RequireCapWithBudget("Clock", ""); err != nil {
		return nil, err
	}

	gameClockMu.Lock()
	dt := gameClock.DeltaTime
	gameClockMu.Unlock()

	return &eval.FloatValue{Value: dt}, nil
}

// _game_total_time: Get total game time in seconds since start
func registerTotalTime() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/game",
		Name:    "_game_total_time",
		NumArgs: 1, // Unit argument
		Effect:  "Clock",
		Type:    makeTotalTimeType,
		Impl:    totalTimeImpl,
		Metadata: &BuiltinMetadata{
			Description: "Get total game time in seconds since start",
			Params:      []ParamDoc{},
			Returns:     "Float representing total game time in seconds",
			Examples: []Example{
				{Code: "_game_total_time()", Description: "Get elapsed game time"},
				{Code: "if _game_total_time() > 60.0 then GameOver else Playing", Description: "Time-based game logic"},
			},
			Since:     "v0.5.1",
			Stability: StabilityStable,
			Tags:      []string{"game", "clock", "time", "total"},
			Category:  "game",
		},
	})
	if err != nil {
		panic("failed to register _game_total_time builtin: " + err.Error())
	}
}

func makeTotalTimeType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.Unit()).Returns(T.Float()).Effects("Clock")
}

func totalTimeImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	if err := ctx.RequireCapWithBudget("Clock", ""); err != nil {
		return nil, err
	}

	gameClockMu.Lock()
	tt := gameClock.TotalTime
	gameClockMu.Unlock()

	return &eval.FloatValue{Value: tt}, nil
}

// _game_frame_count: Get current frame number (starts at 0)
func registerFrameCount() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/game",
		Name:    "_game_frame_count",
		NumArgs: 1, // Unit argument
		Effect:  "Clock",
		Type:    makeFrameCountType,
		Impl:    frameCountImpl,
		Metadata: &BuiltinMetadata{
			Description: "Get current frame number (starts at 0)",
			Params:      []ParamDoc{},
			Returns:     "Int representing the current frame count",
			Examples: []Example{
				{Code: "_game_frame_count()", Description: "Get current frame number"},
				{Code: "if _game_frame_count() % 60 == 0 then doEverySecond()", Description: "Run code every N frames"},
			},
			Since:     "v0.5.1",
			Stability: StabilityStable,
			Tags:      []string{"game", "clock", "frame", "count"},
			Category:  "game",
		},
	})
	if err != nil {
		panic("failed to register _game_frame_count builtin: " + err.Error())
	}
}

func makeFrameCountType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.Unit()).Returns(T.Int()).Effects("Clock")
}

func frameCountImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	if err := ctx.RequireCapWithBudget("Clock", ""); err != nil {
		return nil, err
	}

	gameClockMu.Lock()
	fc := gameClock.FrameCount
	gameClockMu.Unlock()

	return &eval.IntValue{Value: fc}, nil
}
