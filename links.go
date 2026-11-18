package main

import (
	"sync"

	"github.com/couchbase/gocb/v2"
)

// linkedCollections maps each linked component (source ID + link name) to the
// Couchbase collection configured for that link.
//
// Link put/delete callbacks run on the provider's control-interface goroutines
// while document RPCs are served concurrently, so every access goes through
// this type's lock. Callers never see the underlying maps.
type linkedCollections struct {
	mu sync.RWMutex
	// sourceID -> linkName -> collection
	bySource map[string]map[string]*gocb.Collection
}

func newLinkedCollections() *linkedCollections {
	return &linkedCollections{bySource: make(map[string]map[string]*gocb.Collection)}
}

// Put stores (or replaces) the collection for a source/link pair.
func (l *linkedCollections) Put(sourceID, linkName string, collection *gocb.Collection) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.bySource[sourceID] == nil {
		l.bySource[sourceID] = make(map[string]*gocb.Collection)
	}
	l.bySource[sourceID][linkName] = collection
}

// Get returns the collection for a source/link pair, or false if the pair is not linked.
func (l *linkedCollections) Get(sourceID, linkName string) (*gocb.Collection, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	collection := l.bySource[sourceID][linkName]
	return collection, collection != nil
}

// Delete removes a source/link pair, dropping the source entry once it has no links left.
func (l *linkedCollections) Delete(sourceID, linkName string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	links, ok := l.bySource[sourceID]
	if !ok {
		return
	}
	delete(links, linkName)
	if len(links) == 0 {
		delete(l.bySource, sourceID)
	}
}

// Clear removes every link.
func (l *linkedCollections) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	clear(l.bySource)
}
