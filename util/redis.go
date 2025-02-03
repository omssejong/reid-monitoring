package util

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

/**
 * New Redis Client
 * Redis 클라이언트 생성 함수
 *
 * @author: Han Seong San
 * @version: 1.0.0
 * @since: 2024.02.13
 */
func NewRedisClient(addr, pw string) *RedisClient {
	// Redis 클라이언트 설정
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr, // Redis 서버 주소
		Password: pw,   // Redis 서버 비밀번호
		DB:       0,    // 기본 DB 사용
	})

	ctx := context.Background()

	return &RedisClient{rdb, ctx}
}

type RedisClient struct {
	rdb *redis.Client
	ctx context.Context
}

/**
 * Set Redis
 * Redis에 Key, Value, Expiration을 설정하는 함수
 *
 * @author: Han Seong San
 * @version: 1.0.0
 * @since: 2024.02.13
 */
func (r *RedisClient) Set(key string, value interface{}, expiration time.Duration) error {
	return r.rdb.Set(r.ctx, key, value, expiration).Err()
}

/**
 * Get Redis
 * Redis에서 Key에 해당하는 Value를 가져오는 함수
 *
 * @author: Han Seong San
 * @version: 1.0.0
 * @since: 2024.02.13
 */
func (r *RedisClient) Get(key string) (string, error) {
	return r.rdb.Get(r.ctx, key).Result()
}

/**
 * Del Redis
 * Redis에서 Key에 해당하는 Value를 삭제하는 함수
 *
 * @author: Han Seong San
 * @version: 1.0.0
 * @since: 2024.02.13
 */
func (r *RedisClient) Del(key string) error {
	return r.rdb.Del(r.ctx, key).Err()
}

/**
 * Close Redis
 * Redis 연결을 종료하는 함수
 *
 * @author: Han Seong San
 * @version: 1.0.0
 * @since: 2024.02.13
 */
func (r *RedisClient) Close() error {
	return r.rdb.Close()
}

/**
 * Publish Redis
 * Redis에서 Pub/Sub을 사용하여 메시지를 발행하는 함수
 *
 * @author: Han Seong San
 * @version: 1.0.0
 * @since: 2024.03.26
 */
func (r *RedisClient) Publish(channel string, data []byte) error {
	return r.rdb.Publish(r.ctx, channel, data).Err()
}
