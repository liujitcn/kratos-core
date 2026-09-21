package config

import (
	"testing"

	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
)

// TestParseOptionalRedisAndQueue 验证单机配置可以省略 Redis 和队列节点。
func TestParseOptionalRedisAndQueue(t *testing.T) {
	configs := []*configv1.Bootstrap{{}, {Data: &configv1.Data{}}}
	for _, cfg := range configs {
		redisConfig, err := ParseRedis(cfg)
		if err != nil {
			t.Fatalf("ParseRedis() error = %v", err)
		}
		if redisConfig != nil {
			t.Fatalf("ParseRedis() = %v, want nil", redisConfig)
		}

		queueConfig, err := ParseQueue(cfg)
		if err != nil {
			t.Fatalf("ParseQueue() error = %v", err)
		}
		if queueConfig != nil {
			t.Fatalf("ParseQueue() = %v, want nil", queueConfig)
		}
	}
}
