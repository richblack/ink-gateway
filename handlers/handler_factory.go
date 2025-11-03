package handlers

import (
	"log"
	"net/http"
	"semantic-text-processor/services"
	"time"
)

// HandlerFactory creates handlers with proper dependencies
type HandlerFactory struct {
	unifiedService     services.UnifiedChunkService
	legacySupabase     services.SupabaseClient
	tagService         services.TagService
	cacheService       services.CacheService
	logger             *log.Logger
	slowQueryThreshold time.Duration
	metricsEnabled     bool
	useUnifiedHandlers bool
}

// NewHandlerFactory creates a new handler factory
func NewHandlerFactory(
	unifiedService services.UnifiedChunkService,
	legacySupabase services.SupabaseClient,
	tagService services.TagService,
	cacheService services.CacheService,
	logger *log.Logger,
	slowQueryThreshold time.Duration,
	metricsEnabled bool,
	useUnifiedHandlers bool,
) *HandlerFactory {
	return &HandlerFactory{
		unifiedService:     unifiedService,
		legacySupabase:     legacySupabase,
		tagService:         tagService,
		cacheService:       cacheService,
		logger:             logger,
		slowQueryThreshold: slowQueryThreshold,
		metricsEnabled:     metricsEnabled,
		useUnifiedHandlers: useUnifiedHandlers,
	}
}

// ChunkHandlerInterface defines the interface for chunk handlers
type ChunkHandlerInterface interface {
	GetChunks(w http.ResponseWriter, r *http.Request)
	CreateChunk(w http.ResponseWriter, r *http.Request)
	GetChunkByID(w http.ResponseWriter, r *http.Request)
	UpdateChunk(w http.ResponseWriter, r *http.Request)
	DeleteChunk(w http.ResponseWriter, r *http.Request)
	GetChunkChildren(w http.ResponseWriter, r *http.Request)
	GetChunkHierarchy(w http.ResponseWriter, r *http.Request)
	MoveChunk(w http.ResponseWriter, r *http.Request)
}

// TagHandlerInterface defines the interface for tag handlers
type TagHandlerInterface interface {
	AddTag(w http.ResponseWriter, r *http.Request)
	RemoveTag(w http.ResponseWriter, r *http.Request)
	GetChunkTags(w http.ResponseWriter, r *http.Request)
	GetChunksByTag(w http.ResponseWriter, r *http.Request)
}

// CreateChunkHandler creates either a unified or legacy chunk handler
func (f *HandlerFactory) CreateChunkHandler() ChunkHandlerInterface {
	if f.useUnifiedHandlers && f.unifiedService != nil {
		return NewUnifiedChunkHandler(
			f.unifiedService,
			f.cacheService,
			f.logger,
			f.slowQueryThreshold,
			f.metricsEnabled,
		)
	}

	// Return legacy handler wrapped with performance monitoring
	legacy := NewChunkHandler(f.legacySupabase)
	return &LegacyChunkHandlerWrapper{
		Handler:   legacy,
		monitor:   NewPerformanceMonitor(f.slowQueryThreshold, f.logger, f.metricsEnabled),
	}
}

// CreateTagHandler creates either a unified or legacy tag handler
func (f *HandlerFactory) CreateTagHandler() TagHandlerInterface {
	if f.useUnifiedHandlers && f.unifiedService != nil {
		return NewUnifiedTagHandler(
			f.unifiedService,
			f.cacheService,
			f.logger,
			f.slowQueryThreshold,
			f.metricsEnabled,
		)
	}

	// Return legacy handler wrapped with performance monitoring
	legacy := NewTagHandler(f.tagService)
	return &LegacyTagHandlerWrapper{
		Handler: legacy,
		monitor: NewPerformanceMonitor(f.slowQueryThreshold, f.logger, f.metricsEnabled),
	}
}

// LegacyChunkHandlerWrapper wraps the legacy chunk handler with performance monitoring
type LegacyChunkHandlerWrapper struct {
	Handler *ChunkHandler
	monitor *PerformanceMonitor
}

func (w *LegacyChunkHandlerWrapper) GetChunks(rw http.ResponseWriter, r *http.Request) {
	w.monitor.MonitoredHTTPOperation("legacy_get_chunks", rw, func() (int, error) {
		w.Handler.GetChunks(rw, r)
		return http.StatusOK, nil
	})
}

func (w *LegacyChunkHandlerWrapper) CreateChunk(rw http.ResponseWriter, r *http.Request) {
	w.monitor.MonitoredHTTPOperation("legacy_create_chunk", rw, func() (int, error) {
		w.Handler.CreateChunk(rw, r)
		return http.StatusCreated, nil
	})
}

func (w *LegacyChunkHandlerWrapper) GetChunkByID(rw http.ResponseWriter, r *http.Request) {
	w.monitor.MonitoredHTTPOperation("legacy_get_chunk_by_id", rw, func() (int, error) {
		w.Handler.GetChunkByID(rw, r)
		return http.StatusOK, nil
	})
}

func (w *LegacyChunkHandlerWrapper) UpdateChunk(rw http.ResponseWriter, r *http.Request) {
	w.monitor.MonitoredHTTPOperation("legacy_update_chunk", rw, func() (int, error) {
		w.Handler.UpdateChunk(rw, r)
		return http.StatusOK, nil
	})
}

func (w *LegacyChunkHandlerWrapper) DeleteChunk(rw http.ResponseWriter, r *http.Request) {
	w.monitor.MonitoredHTTPOperation("legacy_delete_chunk", rw, func() (int, error) {
		w.Handler.DeleteChunk(rw, r)
		return http.StatusNoContent, nil
	})
}

func (w *LegacyChunkHandlerWrapper) GetChunkChildren(rw http.ResponseWriter, r *http.Request) {
	w.monitor.MonitoredHTTPOperation("legacy_get_chunk_children", rw, func() (int, error) {
		w.Handler.GetChunkChildren(rw, r)
		return http.StatusOK, nil
	})
}

func (w *LegacyChunkHandlerWrapper) GetChunkHierarchy(rw http.ResponseWriter, r *http.Request) {
	w.monitor.MonitoredHTTPOperation("legacy_get_chunk_hierarchy", rw, func() (int, error) {
		w.Handler.GetChunkHierarchy(rw, r)
		return http.StatusOK, nil
	})
}

func (w *LegacyChunkHandlerWrapper) MoveChunk(rw http.ResponseWriter, r *http.Request) {
	w.monitor.MonitoredHTTPOperation("legacy_move_chunk", rw, func() (int, error) {
		w.Handler.MoveChunk(rw, r)
		return http.StatusNoContent, nil
	})
}

// LegacyTagHandlerWrapper wraps the legacy tag handler with performance monitoring
type LegacyTagHandlerWrapper struct {
	Handler *TagHandler
	monitor *PerformanceMonitor
}

func (w *LegacyTagHandlerWrapper) AddTag(rw http.ResponseWriter, r *http.Request) {
	w.monitor.MonitoredHTTPOperation("legacy_add_tag", rw, func() (int, error) {
		w.Handler.AddTag(rw, r)
		return http.StatusCreated, nil
	})
}

func (w *LegacyTagHandlerWrapper) RemoveTag(rw http.ResponseWriter, r *http.Request) {
	w.monitor.MonitoredHTTPOperation("legacy_remove_tag", rw, func() (int, error) {
		w.Handler.RemoveTag(rw, r)
		return http.StatusNoContent, nil
	})
}

func (w *LegacyTagHandlerWrapper) GetChunkTags(rw http.ResponseWriter, r *http.Request) {
	w.monitor.MonitoredHTTPOperation("legacy_get_chunk_tags", rw, func() (int, error) {
		w.Handler.GetChunkTags(rw, r)
		return http.StatusOK, nil
	})
}

func (w *LegacyTagHandlerWrapper) GetChunksByTag(rw http.ResponseWriter, r *http.Request) {
	w.monitor.MonitoredHTTPOperation("legacy_get_chunks_by_tag", rw, func() (int, error) {
		w.Handler.GetChunksByTag(rw, r)
		return http.StatusOK, nil
	})
}