package api

import (
	"context"
	"fmt"
	"sync"
	"time"

	"shrugged/internal/docker"
)

type ContainerPool struct {
	mu         sync.Mutex
	containers []*docker.Container
	config     docker.PostgresConfig
	maxSize    int
	minSize    int
	warming    bool
}

func NewContainerPool(cfg docker.PostgresConfig, minSize, maxSize int) *ContainerPool {
	return &ContainerPool{
		config:  cfg,
		maxSize: maxSize,
		minSize: minSize,
	}
}

func (p *ContainerPool) Start(ctx context.Context) error {
	p.mu.Lock()
	p.warming = true
	p.mu.Unlock()

	for i := 0; i < p.minSize; i++ {
		container, err := docker.StartPostgres(ctx, p.config)
		if err != nil {
			return fmt.Errorf("failed to warm pool: %w", err)
		}

		p.mu.Lock()
		p.containers = append(p.containers, container)
		p.mu.Unlock()
	}

	p.mu.Lock()
	p.warming = false
	p.mu.Unlock()

	go p.maintainPool(ctx)

	return nil
}

func (p *ContainerPool) maintainPool(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.mu.Lock()
			currentSize := len(p.containers)
			needMore := currentSize < p.minSize
			p.mu.Unlock()

			if needMore {
				container, err := docker.StartPostgres(ctx, p.config)
				if err != nil {
					continue
				}

				p.mu.Lock()
				if len(p.containers) < p.maxSize {
					p.containers = append(p.containers, container)
				} else {
					go func() { _ = docker.StopContainer(context.Background(), container.ID) }()
				}
				p.mu.Unlock()
			}
		}
	}
}

func (p *ContainerPool) Acquire(ctx context.Context) (*docker.Container, error) {
	p.mu.Lock()

	if len(p.containers) > 0 {
		container := p.containers[0]
		p.containers = p.containers[1:]
		p.mu.Unlock()

		if err := docker.ResetDatabase(ctx, container); err != nil {
			_ = docker.StopContainer(context.Background(), container.ID)
			return p.Acquire(ctx)
		}

		return container, nil
	}

	p.mu.Unlock()

	return docker.StartPostgres(ctx, p.config)
}

func (p *ContainerPool) Release(container *docker.Container) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.containers) < p.maxSize {
		p.containers = append(p.containers, container)
	} else {
		go func() { _ = docker.StopContainer(context.Background(), container.ID) }()
	}
}

func (p *ContainerPool) Shutdown(ctx context.Context) {
	p.mu.Lock()
	containers := p.containers
	p.containers = nil
	p.mu.Unlock()

	for _, c := range containers {
		_ = docker.StopContainer(ctx, c.ID)
	}
}

func (p *ContainerPool) Size() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.containers)
}
