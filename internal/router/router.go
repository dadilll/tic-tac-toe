package router

import (
	"context"
	"tic_tac_toe/internal/handler"
	"tic_tac_toe/internal/repository"
	"tic_tac_toe/pkg/logger"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

func SetupRoutes(e *echo.Echo, rdb *redis.Client, Logger logger.Logger) {
	// Создаём репозиторий и хендлер
	repo := repository.NewRoomRepository(rdb, context.Background())
	roomHandler := &handler.RoomHandler{
		Repo:   repo,
		Logger: Logger,
	}

	webSocketHandler := &handler.WebSocketHandler{
		Repo:    repo,
		Logger:  Logger,
		Clients: make(map[string][]*websocket.Conn),
	}

	e.POST("/room/create", roomHandler.CreateRoom)
	e.POST("/room/join", roomHandler.JoinRoom)
	e.GET("/room/info/:room_id", roomHandler.GetRoomInfo)
	e.DELETE("/room/delete", roomHandler.DeleteRoom)
	e.POST("/room/delete/user", roomHandler.RemoveUser)
	e.GET("/room/start/:room_id", roomHandler.StartGame)

	e.GET("/ws/:room_id", webSocketHandler.HandleConnection)
}
