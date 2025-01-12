package repository

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
)

type RoomRepository struct {
	rdb *redis.Client
	ctx context.Context
}

func NewRoomRepository(rdb *redis.Client, ctx context.Context) *RoomRepository {
	return &RoomRepository{
		rdb: rdb,
		ctx: ctx,
	}
}

// Создать новую комнату
func (repo *RoomRepository) CreateRoom(roomID, admin string) error {
	roomData := map[string]interface{}{
		"user1":  admin,
		"user2":  "",
		"admin":  admin,
		"status": "waiting",   // waiting, started, finished
		"board":  "         ", // 9 пробелов для пустого поля
		"turn":   admin,       // Хранит, чей сейчас ход
	}

	err := repo.rdb.HSet(repo.ctx, roomID, roomData).Err()
	if err != nil {
		return fmt.Errorf("failed to create room: %w", err)
	}

	return nil
}

// Присоединиться к комнате
func (repo *RoomRepository) JoinRoom(roomID, user string) error {
	room, err := repo.rdb.HGetAll(repo.ctx, roomID).Result()
	if err != nil || len(room) == 0 {
		return fmt.Errorf("room not found")
	}

	if room["user2"] != "" {
		return fmt.Errorf("room is full")
	}

	if room["user1"] == room["user2"] || room["user2"] == room["user1"] {
		return fmt.Errorf("nickname already in use")
	}

	// Присоединяем второго пользователя
	err = repo.rdb.HSet(repo.ctx, roomID, "user2", user).Err()
	if err != nil {
		return fmt.Errorf("failed to join room: %w", err)
	}

	// Если игра уже началась, сразу устанавливаем статус
	if room["status"] == "started" {
		err = repo.rdb.HSet(repo.ctx, roomID, "status", "waiting").Err()
		if err != nil {
			return fmt.Errorf("failed to set room status to waiting: %w", err)
		}
	}

	return nil
}

// Получить информацию о комнате
func (repo *RoomRepository) GetRoomInfo(roomID string) (map[string]string, error) {
	room, err := repo.rdb.HGetAll(repo.ctx, roomID).Result()
	if err != nil || len(room) == 0 {
		return nil, fmt.Errorf("room not found")
	}
	return room, nil
}

// Удалить комнату
func (repo *RoomRepository) DeleteRoom(roomID string) error {
	err := repo.rdb.Del(repo.ctx, roomID).Err()
	if err != nil {
		return fmt.Errorf("failed to delete room: %w", err)
	}
	return nil
}

// Удалить второго пользователя из комнаты
func (repo *RoomRepository) RemoveUser(roomID string) error {
	err := repo.rdb.HSet(repo.ctx, roomID, "user2", "").Err()
	if err != nil {
		return fmt.Errorf("failed to remove user: %w", err)
	}
	return nil
}

// Покинуть комнату
func (repo *RoomRepository) LeaveRoom(roomID, user string) error {
	room, err := repo.rdb.HGetAll(repo.ctx, roomID).Result()
	if err != nil || len(room) == 0 {
		return fmt.Errorf("room not found")
	}

	if room["user1"] == user {
		// Если выходит создатель комнаты, удаляем всю комнату
		return repo.DeleteRoom(roomID)
	} else if room["user2"] == user {
		// Если выходит второй игрок, просто очищаем user2
		return repo.RemoveUser(roomID)
	}

	return fmt.Errorf("user not found in room")
}

// Начать игру
func (repo *RoomRepository) StartGame(roomID string) error {
	fmt.Printf("StartGame called with roomID: %s\n", roomID)

	room, err := repo.rdb.HGetAll(repo.ctx, roomID).Result()
	if err != nil {
		fmt.Printf("Error fetching room from Redis: %s\n", err)
		return fmt.Errorf("room not found")
	}

	if len(room) == 0 {
		return fmt.Errorf("room not found")
	}

	// Проверяем, что user2 есть и что текущий игрок - администратор
	if room["user2"] == "" {
		return fmt.Errorf("cannot start game, room is not full")
	}

	err = repo.rdb.HSet(repo.ctx, roomID, "status", "started").Err()
	if err != nil {
		fmt.Printf("Error updating room status in Redis: %s\n", err)
		return fmt.Errorf("failed to start game: %w", err)
	}

	return nil
}

func (repo *RoomRepository) UpdateRoomField(roomID string, updates map[string]interface{}) error {
	err := repo.rdb.HSet(repo.ctx, roomID, updates).Err()
	if err != nil {
		return fmt.Errorf("failed to update room: %w", err)
	}
	return nil
}
