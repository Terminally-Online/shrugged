package api

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/terminally-online/shrugged/internal/docker"
)

type DatabasePool struct {
	mu        sync.Mutex
	databases []*docker.PoolDatabase
	baseURL   string
	maxSize   int
	minSize   int
	counter   int
}

func NewDatabasePool(minSize, maxSize int) *DatabasePool {
	baseURL := os.Getenv("DATABASE_URL")
	if baseURL == "" {
		baseURL = "postgres://shrugged:shrugged@localhost:5432/shrugged?sslmode=disable"
	}

	return &DatabasePool{
		baseURL: baseURL,
		maxSize: maxSize,
		minSize: minSize,
	}
}

func (p *DatabasePool) Start(ctx context.Context) error {
	for i := 0; i < p.minSize; i++ {
		db, err := p.createDatabase(ctx)
		if err != nil {
			return fmt.Errorf("failed to warm pool: %w", err)
		}

		p.mu.Lock()
		p.databases = append(p.databases, db)
		p.mu.Unlock()
	}

	return nil
}

func (p *DatabasePool) createDatabase(ctx context.Context) (*docker.PoolDatabase, error) {
	p.mu.Lock()
	name := fmt.Sprintf("diff_%d", p.counter)
	p.counter++
	p.mu.Unlock()

	return docker.CreatePoolDatabase(ctx, p.baseURL, name)
}

func (p *DatabasePool) Acquire(ctx context.Context) (*docker.PoolDatabase, error) {
	p.mu.Lock()

	if len(p.databases) > 0 {
		db := p.databases[0]
		p.databases = p.databases[1:]
		p.mu.Unlock()

		if err := docker.ResetPoolDatabase(ctx, db); err != nil {
			_ = docker.DropPoolDatabase(context.Background(), p.baseURL, db.Name)
			return p.Acquire(ctx)
		}

		return db, nil
	}

	p.mu.Unlock()

	return p.createDatabase(ctx)
}

func (p *DatabasePool) Release(db *docker.PoolDatabase) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.databases) < p.maxSize {
		p.databases = append(p.databases, db)
	} else {
		go func() { _ = docker.DropPoolDatabase(context.Background(), p.baseURL, db.Name) }()
	}
}

func (p *DatabasePool) Shutdown(ctx context.Context) {
	p.mu.Lock()
	databases := p.databases
	p.databases = nil
	p.mu.Unlock()

	for _, db := range databases {
		_ = docker.DropPoolDatabase(ctx, p.baseURL, db.Name)
	}
}

func (p *DatabasePool) Size() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.databases)
}
