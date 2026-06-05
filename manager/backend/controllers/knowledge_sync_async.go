package controllers

import (
	"fmt"
	"log"
	"sync"
	"time"

	"xiaozhi/manager/backend/models"

	"gorm.io/gorm"
)

const (
	knowledgeSyncQueueSize   = 256
	knowledgeSyncWorkerCount = 2
)

type knowledgeSyncJobType string

const (
	knowledgeSyncJobUpsert    knowledgeSyncJobType = "upsert"
	knowledgeSyncJobDelete    knowledgeSyncJobType = "delete"
	knowledgeSyncJobDocUpsert knowledgeSyncJobType = "doc_upsert"
	knowledgeSyncJobDocDelete knowledgeSyncJobType = "doc_delete"
)

type knowledgeSyncJob struct {
	jobType           knowledgeSyncJobType
	db                *gorm.DB
	knowledgeBaseID   uint
	documentID        uint
	knowledgeSnapshot *models.KnowledgeBase
	documentSnapshot  *models.KnowledgeBaseDocument
	enqueuedAt        time.Time
}

var (
	knowledgeSyncQueue     chan knowledgeSyncJob
	knowledgeSyncQueueOnce sync.Once
)

func ensureKnowledgeSyncWorkersStarted() {
	knowledgeSyncQueueOnce.Do(func() {
		knowledgeSyncQueue = make(chan knowledgeSyncJob, knowledgeSyncQueueSize)
		for i := 1; i <= knowledgeSyncWorkerCount; i++ {
			go runKnowledgeSyncWorker(i)
		}
		log.Printf("[KnowledgeSync][Async] workers started count=%d queue_size=%d", knowledgeSyncWorkerCount, knowledgeSyncQueueSize)
	})
}

func enqueueKnowledgeSyncUpsert(db *gorm.DB, knowledgeBaseID uint) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	if knowledgeBaseID == 0 {
		return fmt.Errorf("invalid knowledge base ID")
	}
	ensureKnowledgeSyncWorkersStarted()

	job := knowledgeSyncJob{
		jobType:         knowledgeSyncJobUpsert,
		db:              db,
		knowledgeBaseID: knowledgeBaseID,
		enqueuedAt:      time.Now(),
	}
	select {
	case knowledgeSyncQueue <- job:
		log.Printf("[KnowledgeSync][Async] enqueue type=%s kb_id=%d", job.jobType, job.knowledgeBaseID)
		return nil
	default:
		return fmt.Errorf("knowledge base sync queue is full, please retry later")
	}
}

func enqueueKnowledgeSyncDelete(db *gorm.DB, snapshot models.KnowledgeBase) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	ensureKnowledgeSyncWorkersStarted()

	s := snapshot
	job := knowledgeSyncJob{
		jobType:           knowledgeSyncJobDelete,
		db:                db,
		knowledgeBaseID:   snapshot.ID,
		knowledgeSnapshot: &s,
		enqueuedAt:        time.Now(),
	}
	select {
	case knowledgeSyncQueue <- job:
		log.Printf("[KnowledgeSync][Async] enqueue type=%s kb_id=%d", job.jobType, job.knowledgeBaseID)
		return nil
	default:
		return fmt.Errorf("knowledge base sync queue is full, please retry later")
	}
}

func enqueueKnowledgeDocumentSyncUpsert(db *gorm.DB, knowledgeBaseID, documentID uint) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	if knowledgeBaseID == 0 || documentID == 0 {
		return fmt.Errorf("invalid knowledge base or document ID")
	}
	ensureKnowledgeSyncWorkersStarted()

	job := knowledgeSyncJob{
		jobType:         knowledgeSyncJobDocUpsert,
		db:              db,
		knowledgeBaseID: knowledgeBaseID,
		documentID:      documentID,
		enqueuedAt:      time.Now(),
	}
	select {
	case knowledgeSyncQueue <- job:
		log.Printf("[KnowledgeSync][Async] enqueue type=%s kb_id=%d doc_id=%d", job.jobType, job.knowledgeBaseID, job.documentID)
		return nil
	default:
		return fmt.Errorf("knowledge base sync queue is full, please retry later")
	}
}

func enqueueKnowledgeDocumentSyncDelete(db *gorm.DB, kbSnapshot models.KnowledgeBase, docSnapshot models.KnowledgeBaseDocument) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	ensureKnowledgeSyncWorkersStarted()

	kb := kbSnapshot
	doc := docSnapshot
	job := knowledgeSyncJob{
		jobType:           knowledgeSyncJobDocDelete,
		db:                db,
		knowledgeBaseID:   kbSnapshot.ID,
		documentID:        docSnapshot.ID,
		knowledgeSnapshot: &kb,
		documentSnapshot:  &doc,
		enqueuedAt:        time.Now(),
	}
	select {
	case knowledgeSyncQueue <- job:
		log.Printf("[KnowledgeSync][Async] enqueue type=%s kb_id=%d doc_id=%d", job.jobType, job.knowledgeBaseID, job.documentID)
		return nil
	default:
		return fmt.Errorf("knowledge base sync queue is full, please retry later")
	}
}

func runKnowledgeSyncWorker(workerID int) {
	for job := range knowledgeSyncQueue {
		waitMs := time.Since(job.enqueuedAt).Milliseconds()
		start := time.Now()
		switch job.jobType {
		case knowledgeSyncJobUpsert:
			err := processKnowledgeSyncUpsert(job)
			if err != nil {
				log.Printf("[KnowledgeSync][Async] worker=%d type=%s kb_id=%d wait_ms=%d cost_ms=%d err=%v", workerID, job.jobType, job.knowledgeBaseID, waitMs, time.Since(start).Milliseconds(), err)
			} else {
				log.Printf("[KnowledgeSync][Async] worker=%d type=%s kb_id=%d wait_ms=%d cost_ms=%d status=ok", workerID, job.jobType, job.knowledgeBaseID, waitMs, time.Since(start).Milliseconds())
			}
		case knowledgeSyncJobDelete:
			err := processKnowledgeSyncDelete(job)
			if err != nil {
				log.Printf("[KnowledgeSync][Async] worker=%d type=%s kb_id=%d wait_ms=%d cost_ms=%d err=%v", workerID, job.jobType, job.knowledgeBaseID, waitMs, time.Since(start).Milliseconds(), err)
			} else {
				log.Printf("[KnowledgeSync][Async] worker=%d type=%s kb_id=%d wait_ms=%d cost_ms=%d status=ok", workerID, job.jobType, job.knowledgeBaseID, waitMs, time.Since(start).Milliseconds())
			}
		case knowledgeSyncJobDocUpsert:
			err := processKnowledgeDocumentSyncUpsert(job)
			if err != nil {
				log.Printf("[KnowledgeSync][Async] worker=%d type=%s kb_id=%d doc_id=%d wait_ms=%d cost_ms=%d err=%v", workerID, job.jobType, job.knowledgeBaseID, job.documentID, waitMs, time.Since(start).Milliseconds(), err)
			} else {
				log.Printf("[KnowledgeSync][Async] worker=%d type=%s kb_id=%d doc_id=%d wait_ms=%d cost_ms=%d status=ok", workerID, job.jobType, job.knowledgeBaseID, job.documentID, waitMs, time.Since(start).Milliseconds())
			}
		case knowledgeSyncJobDocDelete:
			err := processKnowledgeDocumentSyncDelete(job)
			if err != nil {
				log.Printf("[KnowledgeSync][Async] worker=%d type=%s kb_id=%d doc_id=%d wait_ms=%d cost_ms=%d err=%v", workerID, job.jobType, job.knowledgeBaseID, job.documentID, waitMs, time.Since(start).Milliseconds(), err)
			} else {
				log.Printf("[KnowledgeSync][Async] worker=%d type=%s kb_id=%d doc_id=%d wait_ms=%d cost_ms=%d status=ok", workerID, job.jobType, job.knowledgeBaseID, job.documentID, waitMs, time.Since(start).Milliseconds())
			}
		default:
			log.Printf("[KnowledgeSync][Async] worker=%d unknown_job_type=%s kb_id=%d", workerID, job.jobType, job.knowledgeBaseID)
		}
	}
}

func processKnowledgeSyncUpsert(job knowledgeSyncJob) error {
	if job.db == nil {
		return fmt.Errorf("database connection is nil")
	}
	var kb models.KnowledgeBase
	if err := job.db.Where("id = ?", job.knowledgeBaseID).First(&kb).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return fmt.Errorf("failed to load knowledge base: %w", err)
	}
	return syncKnowledgeBaseBestEffort(job.db, &kb)
}

func processKnowledgeSyncDelete(job knowledgeSyncJob) error {
	if job.db == nil {
		return fmt.Errorf("database connection is nil")
	}
	if job.knowledgeSnapshot == nil {
		return fmt.Errorf("delete sync missing knowledge base snapshot")
	}
	return syncKnowledgeBaseDeleteBestEffort(job.db, job.knowledgeSnapshot)
}

func processKnowledgeDocumentSyncUpsert(job knowledgeSyncJob) error {
	if job.db == nil {
		return fmt.Errorf("database connection is nil")
	}
	return syncKnowledgeDocumentBestEffort(job.db, job.knowledgeBaseID, job.documentID)
}

func processKnowledgeDocumentSyncDelete(job knowledgeSyncJob) error {
	if job.db == nil {
		return fmt.Errorf("database connection is nil")
	}
	if job.knowledgeSnapshot == nil || job.documentSnapshot == nil {
		return fmt.Errorf("document delete sync missing snapshot")
	}
	return syncKnowledgeDocumentDeleteBestEffort(job.db, *job.knowledgeSnapshot, *job.documentSnapshot)
}
