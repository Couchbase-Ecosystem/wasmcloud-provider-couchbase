package main

import (
	"fmt"
	"sync"
	"testing"

	"github.com/couchbase/gocb/v2"
)

func TestLinkedCollectionsPutGetDelete(t *testing.T) {
	links := newLinkedCollections()
	first, second := &gocb.Collection{}, &gocb.Collection{}

	if _, ok := links.Get("component", "default"); ok {
		t.Fatal("expected no collection before any link is stored")
	}

	links.Put("component", "default", first)
	links.Put("component", "analytics", second)

	if got, ok := links.Get("component", "default"); !ok || got != first {
		t.Fatalf("Get(component, default) = %p, %v; want %p, true", got, ok, first)
	}
	if got, ok := links.Get("component", "analytics"); !ok || got != second {
		t.Fatalf("Get(component, analytics) = %p, %v; want %p, true", got, ok, second)
	}
	if _, ok := links.Get("other-component", "default"); ok {
		t.Fatal("expected links to be scoped to their source ID")
	}

	// Re-putting a link replaces its collection.
	links.Put("component", "default", second)
	if got, _ := links.Get("component", "default"); got != second {
		t.Fatalf("expected Put to replace the existing collection")
	}

	links.Delete("component", "default")
	if _, ok := links.Get("component", "default"); ok {
		t.Fatal("expected deleted link to be gone")
	}
	if _, ok := links.Get("component", "analytics"); !ok {
		t.Fatal("deleting one link must not remove the source's other links")
	}

	links.Delete("component", "analytics")
	if _, ok := links.bySource["component"]; ok {
		t.Fatal("expected source entry to be dropped once it has no links")
	}

	// Deleting an unknown link is a no-op.
	links.Delete("missing", "default")
}

func TestLinkedCollectionsClear(t *testing.T) {
	links := newLinkedCollections()
	links.Put("a", "default", &gocb.Collection{})
	links.Put("b", "default", &gocb.Collection{})

	links.Clear()

	if _, ok := links.Get("a", "default"); ok {
		t.Fatal("expected Clear to remove all links")
	}
	if _, ok := links.Get("b", "default"); ok {
		t.Fatal("expected Clear to remove all links")
	}
}

// Link callbacks and document RPCs run on different goroutines. Run with
// `go test -race` to catch unsynchronized access.
func TestLinkedCollectionsConcurrentAccess(t *testing.T) {
	links := newLinkedCollections()
	collection := &gocb.Collection{}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		source := fmt.Sprintf("component-%d", i%3)
		wg.Add(3)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				links.Put(source, "default", collection)
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				links.Get(source, "default")
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				links.Delete(source, "default")
			}
		}()
	}
	wg.Wait()
}
