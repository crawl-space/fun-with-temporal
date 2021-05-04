package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

var ErrNotFound = errors.New("not found")

type PRStatusItem struct {
	State       string    `json:"state"` // should be enum
	TimeStamp   time.Time `json:"timestamp"`
	Description string    `json:"description"`
}

type PRStatus struct {
	Status []PRStatusItem `json:"status"`
}

type Repository struct {
	redis *redis.Client
}

func NewRepository() *Repository {
	r := redis.NewClient(&redis.Options{
		Addr: "redis:6379",
	})

	return &Repository{redis: r}
}

func (r *Repository) Put(ctx context.Context, key string, v interface{}) error {

	o, err := json.Marshal(v)
	if err != nil {
		return err
	}

	return r.redis.Set(ctx, key, o, 0).Err()
}

func (r *Repository) Get(ctx context.Context, key string, v interface{}) error {
	// wrap not found
	in, err := r.redis.Get(ctx, key).Result()
	switch err {
	case nil:
	case redis.Nil:
		return ErrNotFound
	default:
		return fmt.Errorf("error getting value: %w", err)
	}

	return json.Unmarshal([]byte(in), &v)
}
