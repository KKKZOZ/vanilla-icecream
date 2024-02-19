package redis

import (
	"benchmark/ycsb"
	"context"

	"github.com/redis/go-redis/v9"
)

var _ ycsb.DBCreator = (*RedisCreator)(nil)

type RedisCreator struct {
	Rdb *redis.Client
}

func (rc *RedisCreator) Create() (ycsb.DB, error) {
	return NewRedis(rc.Rdb), nil
}

var _ ycsb.DB = (*Redis)(nil)

type Redis struct {
	Rdb *redis.Client
}

func NewRedis(rdb *redis.Client) *Redis {
	return &Redis{
		Rdb: rdb,
	}
}

func (r *Redis) Close() error {
	return nil
}

func (r *Redis) InitThread(ctx context.Context, threadID int, threadCount int) context.Context {
	return ctx
}

func (r *Redis) CleanupThread(ctx context.Context) {
}

func (r *Redis) Read(ctx context.Context, table string, key string) (string, error) {
	keyName := getKeyName(table, key)
	return r.Rdb.Get(context.Background(), keyName).Result()
}

func (r *Redis) Update(ctx context.Context, table string, key string, value string) error {
	keyName := getKeyName(table, key)
	return r.Rdb.Set(context.Background(), keyName, value, 0).Err()
}

func (r *Redis) Insert(ctx context.Context, table string, key string, value string) error {
	keyName := getKeyName(table, key)
	return r.Rdb.Set(context.Background(), keyName, value, 0).Err()
}

func (r *Redis) Delete(ctx context.Context, table string, key string) error {
	keyName := getKeyName(table, key)
	return r.Rdb.Del(context.Background(), keyName).Err()
}

func getKeyName(table string, key string) string {
	return table + "/" + key
}
