package service

import (
	"context"
	"log"
	"sync"
	"time"
)

type OrderWorker struct {
	ordersService *OrdersService
	interval      time.Duration
	workers       int
	mu            sync.RWMutex
	running       bool
	wg            sync.WaitGroup
}

func NewOrderWorker(ordersService *OrdersService, interval time.Duration, workers int) *OrderWorker {
	return &OrderWorker{
		ordersService: ordersService,
		interval:      interval,
		workers:       workers,
		running:       false,
	}
}

func (w *OrderWorker) Start(ctx context.Context) {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		log.Println("Order worker is already running")
		return
	}
	w.running = true
	w.mu.Unlock()

	log.Printf("Starting order worker with %d workers, interval: %v", w.workers, w.interval)

	for i := 0; i < w.workers; i++ {
		w.wg.Add(1)
		go w.worker(ctx, i)
	}
}

func (w *OrderWorker) Stop() {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return
	}
	w.running = false
	w.mu.Unlock()

	w.wg.Wait()
	log.Println("Order worker stopped")
}

func (w *OrderWorker) worker(ctx context.Context, id int) {
	defer w.wg.Done()

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	initialDelay := time.Duration(id) * (w.interval / time.Duration(w.workers*2))
	time.Sleep(initialDelay)

	log.Printf("Worker %d started", id)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d stopped by context", id)
			return
		case <-ticker.C:
			if !w.IsRunning() {
				log.Printf("Worker %d stopped (service not running)", id)
				return
			}

			log.Printf("Worker %d processing orders...", id)
			err := w.ordersService.ProcessPendingOrders()
			if err != nil {
				log.Printf("Worker %d error processing orders: %v", id, err)
			} else {
				log.Printf("Worker %d processed orders successfully", id)
			}
		}
	}
}

func (w *OrderWorker) IsRunning() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.running
}
