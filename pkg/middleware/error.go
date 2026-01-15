package middleware

import (
	"context"
	"fmt"
	"log"
)

// ErrorMiddleware 错误处理中间件，实现自动捕获和错误恢复
type ErrorMiddleware struct {
	next Handler
}

// Handler 处理器接口
type Handler interface {
	Handle(ctx context.Context, input interface{}) (interface{}, error)
}

// NewErrorMiddleware 创建新的错误中间件
func NewErrorMiddleware(next Handler) *ErrorMiddleware {
	return &ErrorMiddleware{
		next: next,
	}
}

// Handle 处理请求，包含错误捕获
func (em *ErrorMiddleware) Handle(ctx context.Context, input interface{}) (interface{}, error) {
	// 执行下一个处理器
	result, err := em.next.Handle(ctx, input)
	if err != nil {
		// 记录错误
		log.Printf("ErrorMiddleware: Captured error - %v", err)

		// 尝试恢复
		if recoverErr := em.recover(ctx, err); recoverErr != nil {
			return nil, fmt.Errorf("failed to recover from error: %w", recoverErr)
		}

		return nil, err
	}

	return result, nil
}

// recover 尝试从错误中恢复
func (em *ErrorMiddleware) recover(ctx context.Context, err error) error {
	// 实现错误恢复逻辑
	// 例如：重试、回滚、清理资源等

	log.Printf("ErrorMiddleware: Attempting recovery from error - %v", err)

	// 根据错误类型执行不同的恢复策略
	switch {
	case isRetryableError(err):
		return em.retry(ctx, err)
	case isCleanupNeeded(err):
		return em.cleanup(ctx, err)
	default:
		return nil
	}
}

// isRetryableError 判断错误是否可重试
func isRetryableError(err error) bool {
	// 实现错误类型判断逻辑
	return false
}

// retry 重试逻辑
func (em *ErrorMiddleware) retry(ctx context.Context, err error) error {
	// 实现重试逻辑
	log.Printf("ErrorMiddleware: Retrying operation")
	return nil
}

// isCleanupNeeded 判断是否需要清理
func isCleanupNeeded(err error) bool {
	// 实现清理判断逻辑
	return false
}

// cleanup 清理逻辑
func (em *ErrorMiddleware) cleanup(ctx context.Context, err error) error {
	// 实现清理逻辑
	log.Printf("ErrorMiddleware: Performing cleanup")
	return nil
}
