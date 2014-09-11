package main

import (
	"fmt"
	"net/http"
)

type ClientPool struct {
	clients chan *http.Client
}

func newClientPool() *ClientPool {
	return &ClientPool{
		clients: make(chan *http.Client, 5),
	}
}

func (p *ClientPool) checkIn(client *http.Client) {
	fmt.Println("Put client into pool")
	p.clients <- client
}

func (p *ClientPool) checkOut() *http.Client {
	select {
	case client := <-p.clients:
		fmt.Println("Procured client from pool")
		return client
	default:
		fmt.Println("Created new client")
		return &http.Client{}
	}
}
