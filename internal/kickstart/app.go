package kickstart

import (
	"os"
	"os/signal"
	"syscall"
)

type KickstartFunc[T any] func(*Context[T]) error

type Context[T any] struct {
	AppHandler T
	Next       LoopState
}

type App[T any] struct {
	initFn      KickstartFunc[T]
	afterInitFn KickstartFunc[T]
	loopFn      KickstartFunc[T]
	afterLoopFn KickstartFunc[T]
}

type AppAfterInit[T any] struct {
	initFn KickstartFunc[T]
	then   KickstartFunc[T]
}

type AppAfterLoop[T any] struct {
	init   *AppAfterInit[T]
	loopFn KickstartFunc[T]
	then   KickstartFunc[T]
}

func Init[T any](initFn KickstartFunc[T]) *AppAfterInit[T] {
	return &AppAfterInit[T]{
		initFn: initFn,
	}
}

func (app *AppAfterInit[T]) Loop(loopFn KickstartFunc[T]) *AppAfterLoop[T] {
	return &AppAfterLoop[T]{
		init:   app,
		loopFn: loopFn,
	}
}

func (app *AppAfterInit[T]) Then(next KickstartFunc[T]) *AppAfterInit[T] {
	app.then = next

	return app
}

func (app *AppAfterInit[T]) Exec() error {
	return exec(&App[T]{
		initFn:      app.initFn,
		afterInitFn: app.then,
	})
}

func (app *AppAfterLoop[T]) Then(next KickstartFunc[T]) *AppAfterLoop[T] {
	app.then = next

	return app
}

func (app *AppAfterLoop[T]) Exec() error {
	return exec(&App[T]{
		initFn:      app.init.initFn,
		afterInitFn: app.init.then,
		loopFn:      app.loopFn,
		afterLoopFn: app.then,
	})
}

type LoopState int

const (
	LoopContinueFlag LoopState = iota
	LoopBreakFlag
)

func exec[T any](app *App[T]) error {
	stopRun := make(chan struct{})
	osSignalCaptor := make(chan os.Signal, 1)
	signal.Notify(osSignalCaptor, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-osSignalCaptor
		close(stopRun)
	}()

	ctx := Context[T]{}

	if err := app.initFn(&ctx); err != nil {
		return err
	}

	if app.afterInitFn != nil {
		if err := app.afterInitFn(&ctx); err != nil {
			return err
		}
	}

LOOP:
	for {
		select {
		case <-stopRun:
			break LOOP
		default:
		}

		err := app.loopFn(&ctx)
		if err != nil {
			return err
		}

		if ctx.Next == LoopContinueFlag {
			continue
		}

		if ctx.Next == LoopBreakFlag {
			break
		}
	}

	if app.afterLoopFn != nil {
		return app.afterLoopFn(&ctx)
	}

	return nil
}
