package eino

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/schema"
	"github.com/sysguard/sysguard/internal/observability"
)

type callbackIDKey struct{}

func NewCallbackBridge(obs *observability.GlobalCallback) callbacks.Handler {
	return callbacks.NewHandlerBuilder().
		OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
			if obs == nil {
				return ctx
			}
			id := obs.OnCallbackStarted(callbackName(info))
			return context.WithValue(ctx, callbackIDKey{}, id)
		}).
		OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
			if obs == nil {
				return ctx
			}
			if id, ok := ctx.Value(callbackIDKey{}).(string); ok && id != "" {
				obs.OnCallbackCompleted(id, map[string]interface{}{
					"component": componentName(info),
					"name":      runInfoName(info),
				})
			}
			return ctx
		}).
		OnErrorFn(func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			if obs == nil {
				return ctx
			}
			if id, ok := ctx.Value(callbackIDKey{}).(string); ok && id != "" {
				obs.OnCallbackError(id, err)
			}
			return ctx
		}).
		OnStartWithStreamInputFn(func(ctx context.Context, info *callbacks.RunInfo, input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
			if input != nil {
				input.Close()
			}
			if obs == nil {
				return ctx
			}
			id := obs.OnCallbackStarted(callbackName(info))
			return context.WithValue(ctx, callbackIDKey{}, id)
		}).
		OnEndWithStreamOutputFn(func(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
			if output != nil {
				output.Close()
			}
			if obs == nil {
				return ctx
			}
			if id, ok := ctx.Value(callbackIDKey{}).(string); ok && id != "" {
				obs.OnCallbackCompleted(id, map[string]interface{}{
					"component": componentName(info),
					"name":      runInfoName(info),
					"stream":    true,
				})
			}
			return ctx
		}).
		Build()
}

func callbackName(info *callbacks.RunInfo) string {
	component := componentName(info)
	name := runInfoName(info)
	if name == "" {
		return "Eino." + component
	}
	return fmt.Sprintf("Eino.%s.%s", component, name)
}

func componentName(info *callbacks.RunInfo) string {
	if info == nil || info.Component == "" {
		return "component"
	}
	return string(info.Component)
}

func runInfoName(info *callbacks.RunInfo) string {
	if info == nil {
		return ""
	}
	if info.Name != "" {
		return info.Name
	}
	return info.Type
}
