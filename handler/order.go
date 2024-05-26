package handler

import (
	"fmt"
	"net/http"
)

type Order struct{}

func (order *Order) Create(w http.ResponseWriter, r * http.Request) {
	fmt.Println("Create an order")
}

func (order *Order) List(w http.ResponseWriter, r * http.Request) {
	fmt.Println("List all order")
} 

func (order *Order) GetById(w http.ResponseWriter, r * http.Request) {
	fmt.Println("Get Order by Id")
}

func (order *Order) UpdateById(w http.ResponseWriter, r * http.Request) {
	fmt.Println("Update Order By Id")
}

func (order *Order) DeleteById(w http.ResponseWriter, r * http.Request) {
	fmt.Println("Update Order By Id")
}