package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"tic_tac_toe/internal/repository"
	"tic_tac_toe/pkg/logger"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WebSocketHandler struct {
	Repo    *repository.RoomRepository
	Logger  logger.Logger
	Mutex   sync.Mutex
	Clients map[string][]*websocket.Conn // Список соединений для каждой комнаты
}

// HandleConnection обрабатывает WebSocket соединение
func (h *WebSocketHandler) HandleConnection(c echo.Context) error {
	roomID := c.Param("room_id")

	if roomID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "room_id is required"})
	}

	roomInfo, err := h.Repo.GetRoomInfo(roomID)
	if err != nil {
		h.Logger.Error(c.Request().Context(), fmt.Sprintf("Room not found: %s", err.Error()))
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Room not found"})
	}

	// Проверяем, что статус комнаты "started"
	if roomInfo["status"] != "started" {
		h.Logger.Error(c.Request().Context(), "Room is not started")
		return echo.NewHTTPError(http.StatusBadRequest, "Room is not started")
	}

	// Проверяем количество соединений
	h.Mutex.Lock()
	if len(h.Clients[roomID]) >= 2 {
		h.Mutex.Unlock()
		h.Logger.Warn(c.Request().Context(), fmt.Sprintf("Room %s already has 2 players connected", roomID))
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Room is full"})
	}
	h.Mutex.Unlock()

	// Обновляем соединение
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		h.Logger.Error(c.Request().Context(), fmt.Sprintf("Failed to upgrade connection: %s", err.Error()))
		return err
	}

	h.Mutex.Lock()
	h.Clients[roomID] = append(h.Clients[roomID], conn)
	h.Mutex.Unlock()

	// Отправляем начальное состояние комнаты новому клиенту
	initialState := map[string]interface{}{
		"type": "initial_state",
		"data": roomInfo,
	}
	if err := conn.WriteJSON(initialState); err != nil {
		h.Logger.Error(c.Request().Context(), fmt.Sprintf("Failed to send initial state to room %s: %s", roomID, err.Error()))
		h.Mutex.Lock()
		h.disconnectRoom(roomID, conn)
		h.Mutex.Unlock()
		conn.Close()
		return err
	}

	defer func() {
		h.Mutex.Lock()
		h.disconnectRoom(roomID, conn)
		h.Mutex.Unlock()
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			h.Logger.Error(c.Request().Context(), fmt.Sprintf("Connection error in room %s: %s", roomID, err.Error()))
			break
		}

		h.Logger.Info(c.Request().Context(), fmt.Sprintf("Message received in room %s: %s", roomID, message))

		var data map[string]string
		if err := json.Unmarshal(message, &data); err != nil {
			h.Logger.Error(c.Request().Context(), fmt.Sprintf("Invalid message format: %s", err.Error()))
			continue
		}

		if data["action"] == "make_move" {
			if err := h.processMove(c.Request().Context(), roomID, data["player"], data["position"]); err != nil {
				h.Logger.Error(c.Request().Context(), fmt.Sprintf("Move error: %s", err.Error()))
				h.BroadcastMessage(c.Request().Context(), roomID, "error", map[string]string{
					"message": err.Error(),
				})
			}
		}
	}
	return nil
}

func (h *WebSocketHandler) processMove(ctx context.Context, roomID, player, position string) error {
	roomInfo, err := h.Repo.GetRoomInfo(roomID)
	if err != nil {
		return fmt.Errorf("failed to fetch room info: %w", err)
	}

	// Проверяем текущего игрока
	if roomInfo["turn"] != player {
		return fmt.Errorf("it's not your turn")
	}

	// Проверяем доску и позицию
	board := roomInfo["board"]
	pos, err := strconv.Atoi(position)
	if err != nil || pos < 0 || pos >= len(board) {
		return fmt.Errorf("invalid position")
	}

	if board[pos] != ' ' {
		return fmt.Errorf("position already occupied")
	}

	// Обновляем доску
	playerSymbol := "X"
	if player == roomInfo["user2"] {
		playerSymbol = "O"
	}
	board = board[:pos] + playerSymbol + board[pos+1:]

	// Проверяем победителя или ничью
	status := "ongoing"
	winner := h.checkWinner(board)
	if winner != "" {
		status = "finished"
	} else if !contains(board, ' ') {
		status = "tie"
	}

	// Обновляем данные в репозитории
	h.Repo.UpdateRoomField(roomID, map[string]interface{}{
		"board":  board,
		"turn":   h.getNextPlayer(roomInfo, player),
		"status": status,
		"winner": winner,
	})

	// Отправляем обновления клиентам
	updatedRoomInfo, err := h.Repo.GetRoomInfo(roomID)
	if err != nil {
		return fmt.Errorf("failed to fetch updated room info: %w", err)
	}

	h.BroadcastMessage(ctx, roomID, "update", updatedRoomInfo)
	return nil
}

// BroadcastMessage отправляет сообщение всем клиентам в комнате
func (h *WebSocketHandler) BroadcastMessage(ctx context.Context, roomID string, messageType string, data interface{}) {
	h.Mutex.Lock()
	defer h.Mutex.Unlock()

	response := map[string]interface{}{
		"type": messageType,
		"data": data,
	}
	responseBytes, err := json.Marshal(response)
	if err != nil {
		h.Logger.Error(ctx, fmt.Sprintf("Failed to marshal response for room %s: %s", roomID, err.Error()))
		return
	}

	connections, exists := h.Clients[roomID]
	if !exists || len(connections) == 0 {
		h.Logger.Warn(ctx, fmt.Sprintf("No active connections found for room %s", roomID))
		return
	}

	// Отправляем сообщение каждому подключенному клиенту
	activeConnections := []*websocket.Conn{}
	for _, conn := range connections {
		if err := conn.WriteMessage(websocket.TextMessage, responseBytes); err != nil {
			h.Logger.Error(ctx, fmt.Sprintf("Failed to send message to room %s: %s", roomID, err.Error()))
			conn.Close()
		} else {
			activeConnections = append(activeConnections, conn)
		}
	}

	// Обновляем список активных соединений
	h.Clients[roomID] = activeConnections
}

func (h *WebSocketHandler) disconnectRoom(roomID string, conn *websocket.Conn) {
	connections, exists := h.Clients[roomID]
	if !exists {
		return
	}

	// Удаляем конкретное соединение
	updatedConnections := []*websocket.Conn{}
	for _, c := range connections {
		if c != conn {
			updatedConnections = append(updatedConnections, c)
		} else {
			// Закрываем соединение
			conn.Close()
		}
	}

	if len(updatedConnections) == 0 {
		// Если соединений больше нет, удаляем комнату из карты
		delete(h.Clients, roomID)
		h.Logger.Info(context.TODO(), fmt.Sprintf("All connections in room %s have been closed", roomID))
	} else {
		h.Clients[roomID] = updatedConnections
	}
}

func (h *WebSocketHandler) checkWinner(board string) string {
	winPatterns := [][]int{
		{0, 1, 2}, {3, 4, 5}, {6, 7, 8}, // горизонтальные линии
		{0, 3, 6}, {1, 4, 7}, {2, 5, 8}, // вертикальные линии
		{0, 4, 8}, {2, 4, 6}, // диагонали
	}

	for _, pattern := range winPatterns {
		if board[pattern[0]] != ' ' && board[pattern[0]] == board[pattern[1]] && board[pattern[1]] == board[pattern[2]] {
			return string(board[pattern[0]])
		}
	}
	return ""
}

func (h *WebSocketHandler) getNextPlayer(roomInfo map[string]string, currentPlayer string) string {
	if currentPlayer == roomInfo["user1"] {
		return roomInfo["user2"]
	}
	return roomInfo["user1"]
}

// Проверка на наличие свободных клеток (для ничьей)
func contains(board string, char rune) bool {
	for _, c := range board {
		if c == char {
			return true
		}
	}
	return false
}
