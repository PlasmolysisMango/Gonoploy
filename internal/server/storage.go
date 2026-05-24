package server

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/PlasmolysisMango/Gonopoly/pkg/protocol"
)

// Storage 接口定义持久化操作。
type Storage interface {
	SaveRoom(roomID string, data *RoomSaveData) error
	LoadRoom(roomID string) (*RoomSaveData, error)
	DeleteRoom(roomID string) error
	ListRooms() ([]string, error)
}

// RoomSaveData 房间持久化数据。
type RoomSaveData struct {
	RoomID    string                `json:"room_id"`
	GameState protocol.S2CGameState `json:"game_state"`
	Players   []RoomPlayerMeta      `json:"players"`
	CreatedAt time.Time             `json:"created_at"`
	UpdatedAt time.Time             `json:"updated_at"`
	Started   bool                  `json:"started"`
}

// RoomPlayerMeta 玩家元数据（含连接信息）。
type RoomPlayerMeta struct {
	Name         string    `json:"name"`
	CharName     string    `json:"char_name"`
	Token        string    `json:"token"`
	Online       bool      `json:"online"`
	DisconnectAt time.Time `json:"disconnect_at,omitempty"`
}

// FileStorage 基于文件系统的存储实现。
type FileStorage struct {
	dir string
	mu  sync.RWMutex
}

// NewFileStorage 创建文件存储，确保目录存在。
func NewFileStorage(dir string) *FileStorage {
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("[storage] failed to create dir %s: %v", dir, err)
	}
	return &FileStorage{dir: dir}
}

// SaveRoom 将房间数据序列化为 JSON 并写入文件（原子写入）。
func (fs *FileStorage) SaveRoom(roomID string, data *RoomSaveData) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	data.UpdatedAt = time.Now()
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	target := filepath.Join(fs.dir, roomID+".json")
	tmp := target + ".tmp"

	if err := os.WriteFile(tmp, raw, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, target)
}

// LoadRoom 从文件读取并反序列化房间数据。
func (fs *FileStorage) LoadRoom(roomID string) (*RoomSaveData, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	target := filepath.Join(fs.dir, roomID+".json")
	raw, err := os.ReadFile(target)
	if err != nil {
		return nil, err
	}

	var data RoomSaveData
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// DeleteRoom 删除房间持久化文件。
func (fs *FileStorage) DeleteRoom(roomID string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	target := filepath.Join(fs.dir, roomID+".json")
	err := os.Remove(target)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// ListRooms 列出目录下所有 .json 文件，返回 roomID 列表。
func (fs *FileStorage) ListRooms() ([]string, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	entries, err := os.ReadDir(fs.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var ids []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".json") && !strings.HasSuffix(name, ".tmp") {
			ids = append(ids, strings.TrimSuffix(name, ".json"))
		}
	}
	return ids, nil
}
