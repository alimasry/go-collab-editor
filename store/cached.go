package store

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/alimasry/go-collab-editor/ot"
)

// dirtyState tracks what needs flushing for a single document.
type dirtyState struct {
	contentDirty bool // content/version needs writing to backing store
	flushedOps   int  // number of ops already flushed (index into history)
	created      bool // doc created locally but not yet in backing store
}

// CachedStore wraps a backing DocumentStore with an in-memory cache.
// All reads and writes are served from the cache. Dirty documents are
// flushed to the backing store periodically in the background.
type CachedStore struct {
	cache         *MemoryStore
	backing       DocumentStore
	mu            sync.Mutex
	dirty         map[string]*dirtyState
	flushInterval time.Duration
	stop          chan struct{}
	done          chan struct{}
}

// NewCachedStore creates a CachedStore that caches in memory and flushes
// dirty documents to the backing store every flushInterval.
func NewCachedStore(backing DocumentStore, flushInterval time.Duration) *CachedStore {
	cs := &CachedStore{
		cache:         NewMemoryStore(),
		backing:       backing,
		dirty:         make(map[string]*dirtyState),
		flushInterval: flushInterval,
		stop:          make(chan struct{}),
		done:          make(chan struct{}),
	}
	go cs.flushLoop()
	return cs
}

func (cs *CachedStore) Create(ctx context.Context, id, content string) error {
	if err := cs.cache.Create(ctx, id, content); err != nil {
		return err
	}
	cs.mu.Lock()
	cs.dirty[id] = &dirtyState{contentDirty: true, created: true}
	cs.mu.Unlock()
	return nil
}

func (cs *CachedStore) Get(ctx context.Context, id string) (*DocumentInfo, error) {
	info, err := cs.cache.Get(ctx, id)
	if err == nil {
		return info, nil
	}
	// Cache miss — load from backing store.
	if err := cs.loadFromBacking(ctx, id); err != nil {
		return nil, err
	}
	return cs.cache.Get(ctx, id)
}

func (cs *CachedStore) List(ctx context.Context) ([]DocumentInfo, error) {
	return cs.backing.List(ctx)
}

func (cs *CachedStore) UpdateContent(ctx context.Context, id, content string, version int) error {
	// Ensure doc is in cache.
	if _, err := cs.Get(ctx, id); err != nil {
		return err
	}
	if err := cs.cache.UpdateContent(ctx, id, content, version); err != nil {
		return err
	}
	cs.mu.Lock()
	ds := cs.dirty[id]
	if ds == nil {
		cs.cache.mu.RLock()
		flushed := len(cs.cache.docs[id].history)
		cs.cache.mu.RUnlock()
		ds = &dirtyState{flushedOps: flushed}
		cs.dirty[id] = ds
	}
	ds.contentDirty = true
	cs.mu.Unlock()
	return nil
}

func (cs *CachedStore) AppendOperation(ctx context.Context, id string, op ot.Operation, version int) error {
	// Ensure doc is in cache.
	if _, err := cs.Get(ctx, id); err != nil {
		return err
	}

	// Snapshot history length before append so we know how many ops were
	// already flushed if this doc was previously clean (removed from dirty map).
	cs.cache.mu.RLock()
	prevLen := len(cs.cache.docs[id].history)
	cs.cache.mu.RUnlock()

	if err := cs.cache.AppendOperation(ctx, id, op, version); err != nil {
		return err
	}
	// Mark dirty so flush loop picks up the new op.
	cs.mu.Lock()
	if cs.dirty[id] == nil {
		cs.dirty[id] = &dirtyState{flushedOps: prevLen}
	}
	cs.mu.Unlock()
	return nil
}

func (cs *CachedStore) GetOperations(ctx context.Context, id string, fromVersion int) ([]ot.Operation, error) {
	// Ensure doc is in cache.
	if _, err := cs.Get(ctx, id); err != nil {
		return nil, err
	}
	return cs.cache.GetOperations(ctx, id, fromVersion)
}

// loadFromBacking loads a document and its operations from the backing store
// into the cache. It sets flushedOps so that already-persisted ops are not
// re-flushed.
func (cs *CachedStore) loadFromBacking(ctx context.Context, id string) error {
	info, err := cs.backing.Get(ctx, id)
	if err != nil {
		return err
	}
	ops, err := cs.backing.GetOperations(ctx, id, 0)
	if err != nil {
		return err
	}

	// Write directly into cache's internal map.
	cs.cache.mu.Lock()
	if _, exists := cs.cache.docs[id]; !exists {
		cs.cache.docs[id] = &docRecord{
			info:    *info,
			history: ops,
		}
	}
	cs.cache.mu.Unlock()

	// Set flushedOps so we don't re-flush existing ops.
	cs.mu.Lock()
	if cs.dirty[id] == nil {
		cs.dirty[id] = &dirtyState{flushedOps: len(ops)}
	}
	cs.mu.Unlock()

	return nil
}

func (cs *CachedStore) flushLoop() {
	ticker := time.NewTicker(cs.flushInterval)
	defer ticker.Stop()
	defer close(cs.done)

	for {
		select {
		case <-ticker.C:
			cs.flush()
		case <-cs.stop:
			cs.flush()
			return
		}
	}
}

// flush writes all dirty documents to the backing store.
func (cs *CachedStore) flush() {
	cs.mu.Lock()
	// Snapshot the dirty map and work on a copy.
	snapshot := make(map[string]*dirtyState, len(cs.dirty))
	for id, ds := range cs.dirty {
		cp := *ds
		snapshot[id] = &cp
	}
	cs.mu.Unlock()

	ctx := context.Background()

	for id, ds := range snapshot {
		// Read current state from cache.
		cs.cache.mu.RLock()
		rec, ok := cs.cache.docs[id]
		if !ok {
			cs.cache.mu.RUnlock()
			continue
		}
		info := rec.info
		totalOps := len(rec.history)
		// Copy the new ops slice while holding the lock.
		var newOps []ot.Operation
		if ds.flushedOps < totalOps {
			newOps = make([]ot.Operation, totalOps-ds.flushedOps)
			copy(newOps, rec.history[ds.flushedOps:])
		}
		cs.cache.mu.RUnlock()

		// 1. Create doc in backing store if needed.
		if ds.created {
			if err := cs.backing.Create(ctx, id, ""); err != nil {
				log.Printf("cached store: failed to create doc %q in backing store: %v", id, err)
				continue
			}
		}

		// 2. Flush new ops (before content, so crash-recovery can replay).
		for i, op := range newOps {
			version := ds.flushedOps + i + 1
			if err := cs.backing.AppendOperation(ctx, id, op, version); err != nil {
				log.Printf("cached store: failed to flush op %d for doc %q: %v", version, id, err)
				// Stop flushing this doc — will retry next cycle.
				break
			}
			ds.flushedOps++
		}

		// 3. Flush content if dirty.
		if ds.contentDirty {
			if err := cs.backing.UpdateContent(ctx, id, info.Content, info.Version); err != nil {
				log.Printf("cached store: failed to flush content for doc %q: %v", id, err)
			} else {
				ds.contentDirty = false
			}
		}

		ds.created = false

		// Update the authoritative dirty state.
		cs.mu.Lock()
		cur := cs.dirty[id]
		if cur != nil {
			cur.flushedOps = ds.flushedOps
			cur.created = ds.created
			// Only clear contentDirty if no new writes happened since snapshot.
			if !ds.contentDirty {
				cur.contentDirty = false
			}
			// Remove from dirty map if fully clean.
			if !cur.contentDirty && !cur.created && cur.flushedOps >= totalOps {
				// Re-check current totalOps — new ops may have arrived.
				cs.cache.mu.RLock()
				if r, ok := cs.cache.docs[id]; ok && cur.flushedOps >= len(r.history) {
					delete(cs.dirty, id)
				}
				cs.cache.mu.RUnlock()
			}
		}
		cs.mu.Unlock()
	}
}

// Close signals the flush loop to perform a final flush and waits for it
// to complete.
func (cs *CachedStore) Close() {
	close(cs.stop)
	<-cs.done
}
