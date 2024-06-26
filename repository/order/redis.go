package order

import (
	"encoding/json"
	"errors"
	"fmt"
	"micro-service/model"
	"time"

	"github.com/go-redis/redis"
)

const txnDelayDuration = time.Hour * 24

type RedisRepo struct {
	Client *redis.Client
}

func orderIdKey(id uint64) string {
	return fmt.Sprintf("order:%d", id)
}

func (r *RedisRepo) Insert(order model.Order) error {
	data, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("failed to encode order: %w", err)
	}

	key := orderIdKey(order.OrderId)

	txn := r.Client.TxPipeline()

	if err := txn.SetNX(key, data, txnDelayDuration).Err(); err != nil {
		txn.Discard()
		return fmt.Errorf("failed to set: %w", err)
	}

	err = txn.SAdd("orders", key).Err()
	if err != nil {
		txn.Discard()
		return fmt.Errorf("failed to add orders to Set: %w", err)
	}

	// Execute the transaction
	_, err = txn.Exec()
	if err != nil {
		return fmt.Errorf("failed to execute transaction: %w", err)
	}

	res, err := r.FindById(order.OrderId)

	if err != nil {
		return fmt.Errorf("failed to find created order: %w", err)
	}

	fmt.Printf("order created: %d", res.OrderId)

	return nil
}

var ErrorNotExists = errors.New("order does not exist")

func (r *RedisRepo) FindById(id uint64) (model.Order, error) {
	key := orderIdKey(id)

	res, err := r.Client.Get(key).Result()
	if errors.Is(err, redis.Nil) {
		return model.Order{}, ErrorNotExists
	} else if err != nil {
		return model.Order{}, fmt.Errorf("get order: %w", err)
	}
	var order model.Order

	err = json.Unmarshal([]byte(res), &order)

	if err != nil {
		return model.Order{}, fmt.Errorf("failed to Unmarshal order %w", err)
	}
	return order, nil
}

func (r *RedisRepo) DeleteById(id uint64) error {
	key := orderIdKey(id)

	txn := r.Client.TxPipeline()

	err := txn.Del(key).Err()

	if errors.Is(err, redis.Nil) {
		txn.Discard()
		return ErrorNotExists
	} else if err != nil {
		txn.Discard()
		return fmt.Errorf("failed to delete order: %w", err)
	}

	if err := txn.SRem("orders", key).Err(); err != nil {
		txn.Discard()
		return fmt.Errorf("failed To Remove from orders Set %w", err)
	}
	_, err = txn.Exec()
	if err != nil {
		return fmt.Errorf("failed to execute transaction: %w", err)
	}
	return nil
}

func (r *RedisRepo) Update(order model.Order) error {

	data, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("failed to encode order: %w", err)
	}

	key := orderIdKey(order.OrderId)

	err = r.Client.SetXX(key, string(data), txnDelayDuration).Err()

	if errors.Is(err, redis.Nil) {
		return ErrorNotExists
	} else if err != nil {
		return fmt.Errorf("failed to update: %w", err)
	}

	return nil
}

type FindAllPage struct {
	Size   int64
	Offset uint64
}

type FindResult struct {
	Orders []model.Order
	Cursor uint64
}

func (r *RedisRepo) FindAll(page FindAllPage) (FindResult, error) {
	res := r.Client.SScan("orders", page.Offset, "*", page.Size)

	keys, cursor, err := res.Result()

	if err != nil {
		return FindResult{}, fmt.Errorf("failed to get order Id's: %w", err)
	}

	if len(keys) == 0 {
		return FindResult{
			Orders: []model.Order{},
		}, nil
	}

	xs, err := r.Client.MGet(keys...).Result()
	if err != nil || xs == nil || len(xs) == 0 {
		return FindResult{}, fmt.Errorf("failed to get orders: %w", err)
	}

	orders := make([]model.Order, len(xs))

	for i, x := range xs {
		x := x.(string)
		var order model.Order

		err := json.Unmarshal([]byte(x), &order)
		if err != nil {
			return FindResult{}, fmt.Errorf("failed to Unmarshal order %w", err)
		}

		orders[i] = order
	}

	return FindResult{
		Orders: orders,
		Cursor: cursor,
	}, nil

}
