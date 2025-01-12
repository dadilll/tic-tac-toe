package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"tic_tac_toe/internal/repository"
	"tic_tac_toe/pkg/logger"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type RoomHandler struct {
	Repo   *repository.RoomRepository
	Logger logger.Logger
}

// CustomValidator связывает Echo с библиотекой валидации
type CustomValidator struct {
	Validator *validator.Validate
}

// Validate вызывает метод Struct библиотеки validator
func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.Validator.Struct(i)
}

// Унифицированная функция для обработки ответов
func (h *RoomHandler) respond(c echo.Context, status int, data interface{}) error {
	dataStr, err := json.Marshal(data) // Преобразуем данные в строку JSON
	if err != nil {
		h.Logger.Error(c.Request().Context(), "Failed to marshal response data: "+err.Error())
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}

	h.Logger.Info(c.Request().Context(), "Response: "+string(dataStr)) // Логируем данные как строку
	return c.JSON(status, data)
}

// Создать комнату
func (h *RoomHandler) CreateRoom(c echo.Context) error {
	type request struct {
		Admin string `json:"admin" validate:"required"`
	}

	var req request
	if err := c.Bind(&req); err != nil {
		h.Logger.Error(c.Request().Context(), "Failed to parse request: "+err.Error())
		return h.respond(c, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	// Валидация данных
	if err := c.Validate(&req); err != nil {
		h.Logger.Error(c.Request().Context(), "Validation error: "+err.Error())
		return h.respond(c, http.StatusBadRequest, map[string]string{
			"error":  "Validation failed",
			"detail": err.Error(),
		})
	}
	// Уникальный ID комнаты
	roomID := "room-" + uuid.New().String()
	err := h.Repo.CreateRoom(roomID, req.Admin)
	if err != nil {
		h.Logger.Error(c.Request().Context(), "Failed to create room: "+err.Error())
		return h.respond(c, http.StatusInternalServerError, map[string]string{"error": "Failed to create room"})
	}

	return h.respond(c, http.StatusOK, map[string]interface{}{
		"roomID": roomID,
		"user1":  req.Admin,
		"user2":  nil,
	})
}

// Присоединиться к комнате
func (h *RoomHandler) JoinRoom(c echo.Context) error {
	type request struct {
		RoomID string `json:"roomID" validate:"required"`
		User   string `json:"user" validate:"required"`
	}

	var req request
	if err := c.Bind(&req); err != nil {
		h.Logger.Error(c.Request().Context(), "Failed to parse request: "+err.Error())
		return h.respond(c, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	// Валидация данных
	if err := c.Validate(&req); err != nil {
		h.Logger.Error(c.Request().Context(), "Validation error: "+err.Error())
		return h.respond(c, http.StatusBadRequest, map[string]string{"error": "Validation failed"})
	}

	err := h.Repo.JoinRoom(req.RoomID, req.User)
	if err != nil {
		h.Logger.Error(c.Request().Context(), "Failed to join room: "+err.Error())
		return h.respond(c, http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	roomInfo, _ := h.Repo.GetRoomInfo(req.RoomID) // Игнорируем ошибку, так как комната должна существовать

	return h.respond(c, http.StatusOK, map[string]interface{}{
		"roomID": req.RoomID,
		"user1":  roomInfo["user1"],
		"user2":  roomInfo["user2"],
	})
}

// Получить информацию о комнате
func (h *RoomHandler) GetRoomInfo(c echo.Context) error {
	roomID := c.Param("room_id")

	// Логируем roomID для проверки
	h.Logger.Info(c.Request().Context(), fmt.Sprintf("Fetching room info for roomID: %s", roomID))

	roomInfo, err := h.Repo.GetRoomInfo(roomID)
	if err != nil {
		// Логируем ошибку, если комната не найдена
		h.Logger.Error(c.Request().Context(), "Failed to get room info: "+err.Error())
		return h.respond(c, http.StatusNotFound, map[string]string{"error": "Room not found"})
	}

	// Логируем успешное извлечение информации
	h.Logger.Info(c.Request().Context(), fmt.Sprintf("Room info fetched successfully for roomID: %s", roomID))
	return h.respond(c, http.StatusOK, roomInfo)
}

// Удалить комнату
func (h *RoomHandler) DeleteRoom(c echo.Context) error {
	type request struct {
		RoomID string `json:"roomID" validate:"required"`
		Admin  string `json:"admin" validate:"required"`
	}

	var req request
	if err := c.Bind(&req); err != nil {
		h.Logger.Error(c.Request().Context(), "Failed to parse request: "+err.Error())
		return h.respond(c, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	// Валидация данных
	if err := c.Validate(&req); err != nil {
		h.Logger.Error(c.Request().Context(), "Validation error: "+err.Error())
		return h.respond(c, http.StatusBadRequest, map[string]string{"error": "Validation failed"})
	}

	roomInfo, err := h.Repo.GetRoomInfo(req.RoomID)
	if err != nil || roomInfo["admin"] != req.Admin {
		h.Logger.Error(c.Request().Context(), "Unauthorized delete attempt")
		return h.respond(c, http.StatusForbidden, map[string]string{"error": "Only the admin can delete the room"})
	}

	err = h.Repo.DeleteRoom(req.RoomID)
	if err != nil {
		h.Logger.Error(c.Request().Context(), "Failed to delete room: "+err.Error())
		return h.respond(c, http.StatusInternalServerError, map[string]string{"error": "Failed to delete room"})
	}

	return h.respond(c, http.StatusOK, map[string]string{"message": "Room deleted successfully"})
}

// Удалить второго пользователя из комнаты
func (h *RoomHandler) RemoveUser(c echo.Context) error {
	type request struct {
		RoomID string `json:"roomID" validate:"required"`
		Admin  string `json:"admin" validate:"required"`
	}

	var req request
	if err := c.Bind(&req); err != nil {
		h.Logger.Error(c.Request().Context(), "Failed to parse request: "+err.Error())
		return h.respond(c, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	// Валидация данных
	if err := c.Validate(&req); err != nil {
		h.Logger.Error(c.Request().Context(), "Validation error: "+err.Error())
		return h.respond(c, http.StatusBadRequest, map[string]string{"error": "Validation failed"})
	}

	roomInfo, err := h.Repo.GetRoomInfo(req.RoomID)
	if err != nil || roomInfo["admin"] != req.Admin {
		h.Logger.Error(c.Request().Context(), "Unauthorized remove attempt")
		return h.respond(c, http.StatusForbidden, map[string]string{"error": "Only the admin can remove a user"})
	}

	err = h.Repo.RemoveUser(req.RoomID)
	if err != nil {
		h.Logger.Error(c.Request().Context(), "Failed to remove user: "+err.Error())
		return h.respond(c, http.StatusInternalServerError, map[string]string{"error": "Failed to remove user"})
	}

	return h.respond(c, http.StatusOK, map[string]string{"message": "User removed successfully"})
}

// Начать игру
func (h *RoomHandler) StartGame(c echo.Context) error {
	roomID := c.Param("room_id")

	// Пытаемся начать игру
	err := h.Repo.StartGame(roomID)
	if err != nil {
		h.Logger.Error(c.Request().Context(), "Failed to start game: "+err.Error())
		return h.respond(c, http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// Игра началась, теперь можно отправить сообщение всем игрокам
	return h.respond(c, http.StatusOK, map[string]string{"message": "Game started"})
}
