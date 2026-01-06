package amp

import (
	"database/sql"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// PendingCleaner 定期清理超时的 pending 请求记录
type PendingCleaner struct {
	db       *sql.DB
	interval time.Duration
	timeout  time.Duration
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewPendingCleaner 创建 pending 记录清理器
func NewPendingCleaner(db *sql.DB) *PendingCleaner {
	return &PendingCleaner{
		db:       db,
		interval: 5 * time.Minute,
		timeout:  10 * time.Minute,
		stopChan: make(chan struct{}),
	}
}

// Start 启动后台清理 goroutine
func (c *PendingCleaner) Start() {
	c.wg.Add(1)
	go c.run()
}

// Stop 优雅停止清理器
func (c *PendingCleaner) Stop() {
	close(c.stopChan)
	c.wg.Wait()
}

func (c *PendingCleaner) run() {
	defer c.wg.Done()
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	c.cleanup()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopChan:
			return
		}
	}
}

func (c *PendingCleaner) cleanup() {
	cutoff := time.Now().Add(-c.timeout)
	result, err := c.db.Exec(`
		UPDATE request_logs 
		SET status = 'error', 
			error_type = 'timeout_cleanup',
			updated_at = CURRENT_TIMESTAMP
		WHERE status = 'pending' AND created_at < ?
	`, cutoff)

	if err != nil {
		log.Errorf("pending cleaner: cleanup failed: %v", err)
		return
	}

	if rows, _ := result.RowsAffected(); rows > 0 {
		log.Infof("pending cleaner: cleaned up %d stale pending requests", rows)
	}
}

var globalPendingCleaner *PendingCleaner

// InitPendingCleaner 初始化并启动全局 pending 清理器
func InitPendingCleaner(db *sql.DB) {
	globalPendingCleaner = NewPendingCleaner(db)
	globalPendingCleaner.Start()
	log.Info("pending cleaner: started")
}

// StopPendingCleaner 停止全局 pending 清理器
func StopPendingCleaner() {
	if globalPendingCleaner != nil {
		globalPendingCleaner.Stop()
		log.Info("pending cleaner: stopped")
	}
}
