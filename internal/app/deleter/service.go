package deleter

import (
	"fmt"
	"inmemoryStorageString/internal/app/storage"
	"time"
)

type Service struct {
	interval time.Duration
	stop     chan bool
	cache    *storage.Service
}

func New(duration time.Duration, stop chan bool, cache *storage.Service) *Service {
	return &Service{
		interval: duration,
		stop:     stop,
		cache:    cache,
	}
}

func (s *Service) Run() {
	ticker := time.NewTicker(s.interval)
	for {
		select {
		case <-ticker.C:
			s.cache.DeleteExpired()
		case <-s.stop:
			ticker.Stop()
			return
		}
	}
}

func (s *Service) Stop() {
	fmt.Println("deleter stop")
	s.stop <- true
}
