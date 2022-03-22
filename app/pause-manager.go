package app

import (
	"time"

	"github.com/FreeFeed/freefeed-tg-client/types"
	"github.com/davidmz/debug-log"
)

type PauseManagerCfg struct {
	interval        time.Duration
	cleanupInterval time.Duration
	onResume        func(types.TgChatID)
	closeChan       <-chan struct{}
	debugLogger     debug.Logger
}

type PauseManager struct {
	PauseManagerCfg
	times      map[types.TgChatID]time.Time
	pauseChan  chan types.TgChatID
	resumeChan chan types.TgChatID
}

func NewPauseManager(cfg PauseManagerCfg) *PauseManager {
	p := &PauseManager{
		PauseManagerCfg: cfg,
		times:           make(map[types.TgChatID]time.Time),
		pauseChan:       make(chan types.TgChatID),
		resumeChan:      make(chan types.TgChatID),
	}
	p.debugLogger = p.debugLogger.Fork(p.debugLogger.Name() + ":pauseManager")
	go p.loop()
	return p
}

func (p *PauseManager) IsPaused(id types.TgChatID) bool { _, ok := p.times[id]; return ok }
func (p *PauseManager) Pause(id types.TgChatID)         { p.pauseChan <- id }
func (p *PauseManager) Resume(id types.TgChatID)        { p.resumeChan <- id }

func (p *PauseManager) loop() {
	p.debugLogger.Println("▶️ Starting pause manager")
	defer p.debugLogger.Println("⏹️ Stopping pause manager")

	ticker := time.NewTicker(p.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case id := <-p.pauseChan:
			p.debugLogger.Println("Pausing", id)
			p.times[id] = time.Now()
		case id := <-p.resumeChan:
			p.debugLogger.Println("Resuming", id)
			p._resume(id)
		case <-ticker.C:
			for id, t := range p.times {
				if time.Since(t) >= p.interval {
					p.debugLogger.Println("Cleaning up and resuming", id)
					p._resume(id)
				}
			}
		case <-p.closeChan:
			return
		}
	}
}

func (p *PauseManager) _resume(id types.TgChatID) {
	delete(p.times, id)
	// Run it in background to prevent any surprises
	go p.onResume(id)
}
