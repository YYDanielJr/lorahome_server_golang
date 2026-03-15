package TokenManager

import (
	"container/list"
	"fmt"
	"log"
	"sync"
	"time"
)

type TokenItem struct {
	token    string
	username string
	expires  int64
}

// TokenManager 管理用户token
type TokenManager struct {
	tokenItemList *list.List
	mu            sync.RWMutex
}

func NewTokenManager() *TokenManager {
	tm := &TokenManager{
		tokenItemList: list.New(),
	}
	// 启动清理过期token的goroutine
	go tm.cleanupExpiredTokens()
	return tm
}

func (tm *TokenManager) GenerateToken(username string) string {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	token := fmt.Sprintf("token_%d_%s", time.Now().UnixNano(), username)
	expireTime := time.Now().Add(24 * time.Hour).Unix()

	item := &TokenItem{token, username, expireTime}

	tm.tokenItemList.PushBack(item)
	return token
}

func (tm *TokenManager) ValidateToken(token string) (string, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	now := time.Now().Unix()
	for e := tm.tokenItemList.Front(); e != nil; e = e.Next() {
		item := e.Value.(*TokenItem)
		if item.token == token {
			// 检查是否过期
			if now > item.expires {
				return "", false
			}
			return item.username, true
		}
	}
	return "", false
}

func (tm *TokenManager) RemoveTokenForUser(username string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	var toRemove []*list.Element

	// 先收集要删除的元素
	for e := tm.tokenItemList.Front(); e != nil; e = e.Next() {
		item := e.Value.(*TokenItem)
		if item.username == username {
			toRemove = append(toRemove, e)
		}
	}

	// 批量删除
	for _, e := range toRemove {
		tm.tokenItemList.Remove(e)
		log.Printf("Removed token for user: %s", username)
	}
}

func (tm *TokenManager) cleanupExpiredTokens() {
	for {
		time.Sleep(5 * time.Minute)
		tm.mu.Lock()

		now := time.Now().Unix()
		var toRemove []*list.Element

		// 收集所有过期的token
		for e := tm.tokenItemList.Front(); e != nil; e = e.Next() {
			item := e.Value.(*TokenItem)
			if now > item.expires {
				toRemove = append(toRemove, e)
				log.Printf("Marked expired token for user: %s (expired at: %d, now: %d)",
					item.username, item.expires, now)
			}
		}

		// 批量删除过期的token
		for _, e := range toRemove {
			tm.tokenItemList.Remove(e)
		}

		log.Printf("Cleaned up %d expired tokens", len(toRemove))
		tm.mu.Unlock()
	}
}
