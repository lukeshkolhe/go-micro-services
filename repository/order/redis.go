package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"micro-service/model"

	"github.com/redis/go-redis"
)

type RedisRepo struct {
	Client *redis.Client
}

func orderIdKey(id uint64) string {
	return fmt.Sprintf("order:%d", id)
}

func (r * RedisRepo) Insert(ctx context.Context, order model.Order) error {
	data, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("failed to encode order: %w", err)
	}

	key := orderIdKey(order.OrderId)

	txn: r.Client.TxPipeline()

	if err := txn.SetNX(ctx, key, string(data), 0).Err(); err != nil {
		txn.Discard()
		return fmt.Errorf("failed to set: %w", err)
	}

	if err := txn.SAdd(ctx, "orders", key).Err(); err != nil {
		txn.Discard()
		return fmt.Errorf("failed to add orders to Set: %w", err)
	}

	return nil
}

var ErrorNotExists = errors.New("order does not exist")

func (r * RedisRepo) FindById(ctx context.Context, id uint64) (model.Order, error) {
	key := orderIdKey(id)

	value, err := r.Client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil)) {
		return model.Order{}, ErrorNotExists
	} else if(err != nil) {
		return model.Order{}, fmt.Errorf("get order: %w", &err)
	}
}

func (r * RedisRepo) DeleteById(ctx context.Context, id uint64) error {
	key := orderIdKey(id)

	txn: r.Client.TxPipeline()

	err := txn.Del(ctx, key).Err()

	if errors.Is(err, redis.Nil)) {
		txn.Discard()
		return ErrorNotExists
	} else if(err != nil) {
		txn.Discard()
		return fmt.Errorf("Delete order: %w", &err)
	}

	if(err := txn.SRem(ctx, "orders", key).Err(); err != nil) {
		txn.Discard()
		return fmt.Errorf("Failed To Remove from orders Set %w", &err)
	}

	return nil
}

func (r * RedisRepo) Update(ctx context.Context, model.Order order) error {

	data, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("failed to encode order: %w", err)
	}

	key := orderIdKey(order.OrderId)

	res := r.Client.SetXX(ctx, key, string(data), 0).Err()

	if errors.Is(err, redis.Nil)) {
		return ErrorNotExists
	} else if err != nil {
		return fmt.Errorf("failed to update: %w", err)
	}

	return nil
}

type FindAllPage struct {
	Size uint
	Offset uint
}

type FindResult struct {
	Orders []mode.Order
	Cursor uint
}

func (r * RedisRepo) FindAll(ctx context.Context, page FindAllPage) ([]model.Order, error) {
	res := r.Client.SScan(ctx, "orders", page.Offset, "*", int64(page.Size))

	keys, cursor, err := res.Result()

	if err != nil {
		return FindResult{}, fmt.Errorf("failed to get order Id's: %w", err)
	}

	if len(keys) == 0 {
		return FindResult{
			Orders: []model.Order{},
		}, nil
	}

	xs, err := r.Client.MGet(ctx, keys...).Result()
	if err != nil {
		return FindResult{}, fmt.Errorf("failed to get orders: %w", err)
	}

	orders := make([]model.Order, len(xs))

	for i, x := range xs {
		x := x.(string)
		var order model.Order

		err := json.Unmarshal([] byte x, &order)
		if err != nil {
			return fmt.Errorf("failed to Unmarshal order %w", err)
		}

		orders[i] = order
	}

	return FindResult {
		Orders: orders,
		Cursor: cursor,
	}, nil

}