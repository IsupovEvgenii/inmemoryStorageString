package deleter

import (
	"fmt"
	"inmemoryStorageString/internal/app/storage"
	"time"
)

type Service struct {
	stop      chan bool
	cache     *storage.Service
	intervals chan time.Duration
}

func New(stop chan bool, cache *storage.Service) *Service {
	return &Service{
		stop:  stop,
		cache: cache,
	}
}

func (s *Service) Run() {
	for {
		interval := <-s.intervals
		ticker := time.NewTicker(interval)
		for {
			select {
			case <-ticker.C:
				s.cache.DeleteExpired()
				break
			case <-s.stop:
				ticker.Stop()
				return
			}
		}
	}
}

func (s *Service) Stop() {
	fmt.Println("deleter stop")
	s.stop <- true
}
