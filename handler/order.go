package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi"
	"math/rand"
	"micro-service/model"
	"micro-service/repository/order"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type Order struct {
	Repo *order.RedisRepo
}

func (h *Order) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		CustomerID uuid.UUID        `json:"customer_id"`
		LineItems  []model.LineItem `json:"line_items"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()

	ord := model.Order{
		OrderId:    rand.Uint64(),
		CustemerId: body.CustomerID,
		LineItems:  body.LineItems,
		CreatedAt:  &now,
	}

	err := h.Repo.Insert(ord)
	if err != nil {
		fmt.Println("failed to insert:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	res, err := json.Marshal(ord)
	if err != nil {
		fmt.Println("failed to marshal:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err2 := w.Write(res)
	if err2 != nil {
		fmt.Println("failed to write:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	return
}

func (h *Order) List(w http.ResponseWriter, r *http.Request) {
	cursorStr := r.URL.Query().Get("cursor")
	if cursorStr == "" {
		cursorStr = "0"
	}

	const decimal = 10
	const bitSize = 64
	cursor, err := strconv.ParseUint(cursorStr, decimal, bitSize)
	if err != nil {
		fmt.Println("failed to parse cursor:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	const size = 50
	res, err := h.Repo.FindAll(order.FindAllPage{
		Offset: cursor,
		Size:   size,
	})
	if err != nil {
		fmt.Println("failed to get orders:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var response struct {
		Items []model.Order `json:"items"`
		Next  uint64        `json:"next,omitempty"`
	}
	response.Items = res.Orders
	response.Next = res.Cursor

	data, err := json.Marshal(response)
	if err != nil {
		fmt.Println("failed to marshal:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = w.Write(data)
	if err != nil {
		fmt.Println("failed to marshal:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	return
}

func (h *Order) GetById(w http.ResponseWriter, r *http.Request) {
	idParam, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		fmt.Println("failed to parse id:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	res, err := h.Repo.FindById(uint64(idParam))
	if errors.Is(err, order.ErrorNotExists) {
		fmt.Println("order does not exist", err)
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		fmt.Println("failed to get order:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(res)
	if err != nil {
		fmt.Println("failed to marshal:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = w.Write(data)
	if err != nil {
		fmt.Println("failed to marshal:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	return
}

func (h *Order) UpdateById(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		fmt.Println("failed to parse request body:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	idParam, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		fmt.Println("failed to parse order id", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	o, err := h.Repo.FindById(uint64(idParam))
	if errors.Is(err, order.ErrorNotExists) {
		fmt.Println("order does not exist", err)
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		fmt.Println("failed to find order", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	const completedStatus = "completed"
	const shippedStatus = "shipped"
	now := time.Now().UTC()

	switch body.Status {
	case shippedStatus:
		if o.ShippedAt != nil {
			fmt.Println("Order is already shipped:", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		o.OrderStatus = shippedStatus
		o.ShippedAt = &now

	case completedStatus:
		if o.CompletedAt != nil {
			fmt.Println("Order is already completed:", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		} else if o.ShippedAt == nil {
			fmt.Println("Order is not yet shipped:", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		o.OrderStatus = completedStatus
		o.CompletedAt = &now

	default:
		fmt.Println("incorrect status:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = h.Repo.Update(o)
	if err != nil {
		fmt.Println("failed to update:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(o)
	if err != nil {
		fmt.Println("failed to marshal:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = w.Write(data)
	if err != nil {
		fmt.Println("failed to write:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Order) DeleteById(w http.ResponseWriter, r *http.Request) {
	idParam, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		fmt.Println("failed to parse id:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = h.Repo.DeleteById(idParam)
	if err != nil {
		fmt.Println("failed to delete order:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
