package tg

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
)

var (
	DeafultHashCollector *HashCollector
)

type HashCollector struct {
	db        *sqlx.DB
	hashes    []string
	collector chan string
	stopCh    chan struct{}
	mut       sync.Mutex
}

func NewHashCollector(db *sqlx.DB) *HashCollector {
	hc := &HashCollector{
		db:        db,
		hashes:    make([]string, 0),
		collector: make(chan string, 100),
		stopCh:    make(chan struct{}, 0),
		mut:       sync.Mutex{},
	}

	DeafultHashCollector = hc
	return hc
}

func (h *HashCollector) Collect(in string) {
	h.collector <- in
}

func (h *HashCollector) Start() {
	storeTicker := time.NewTicker(time.Second)
	defer func() {
		storeTicker.Stop()
		close(h.stopCh)
		close(h.collector)
		log.Println("HashCollector stopped")
	}()
	defer func() {
		h.mut.Lock()
		h.Store()
		h.mut.Unlock()
	}()

	for {
		select {
		case <-h.stopCh:
			return
		case <-storeTicker.C:
			h.mut.Lock()
			if len(h.hashes) >= 20 {
				h.Store()
			}
			h.mut.Unlock()
		case in := <-h.collector:
			h.mut.Lock()
			h.hashes = append(h.hashes, getHashes(in)...)
			h.mut.Unlock()
		}
	}
}

func (h *HashCollector) Stop() {
	h.stopCh <- struct{}{}
}

func (h *HashCollector) Store() {

	args := make([]interface{}, 0)
	valueItems := make([]string, 0)
	for _, h := range h.hashes {
		valueItems = append(valueItems, "(?)")
		args = append(args, h)
	}
	query := fmt.Sprintf("insert ignore into hashes (hash) values %s", strings.Join(valueItems, ","))

	_, err := h.db.Exec(query, args...)
	if err != nil {
		log.Println("collectHashes", err)
	}

	// clear hashes
	h.hashes = make([]string, 0)
}
