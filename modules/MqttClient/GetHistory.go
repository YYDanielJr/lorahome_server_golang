package mqttclient

import (
	"encoding/json"
	"log"
	"strings"
	"time"

	store "lorahome_server/modules/Store"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// ============ 请求/响应结构体 ============

// HistoryRequest 客户端请求历史数据
type HistoryRequest struct {
	DevEui    string `json:"dev_eui"`
	Limit     int    `json:"limit"`
	SinceTime string `json:"since_time,omitempty"` // 可选: "2024-03-15T10:00:00Z"
}

// HistoryResponse 返回给客户端的响应
type HistoryResponse struct {
	Success bool                `json:"success"`
	Message string              `json:"message,omitempty"`
	Data    []store.HistoryItem `json:"data,omitempty"`
}

// HistoryItem 统一的历史数据项（复用 Store 中的定义或在此声明）
// type HistoryItem struct {
// 	ReceiveTime time.Time `json:"receive_time"`
// 	Type        int       `json:"type"`              // 1=温度,2=湿度,-1=灯控制
// 	Value       float64   `json:"value,omitempty"`   // 温湿度值
// 	IsOpen      string    `json:"is_open,omitempty"` // "1"/"0" 灯状态
// }

// ============ MQTT 请求处理函数 ============

// handleGetControlHistory 解析 topic 并处理请求
func (r *MqttClient) handleGetControlHistory(client mqtt.Client, msg mqtt.Message) {
	topic := msg.Topic()
	log.Println("Received GetControlHistory request.")
	parts := strings.Split(topic, "/")
	// 格式: userid/client/home/homeid/room/roomid/device/deveui/GetControlHistoryRequest
	if len(parts) < 9 {
		r.publishHistoryResponse(client, topic, false, "invalid topic", nil)
		return
	}
	userID, _, _, devEui := parts[0], parts[3], parts[5], parts[7]

	var req HistoryRequest
	if err := json.Unmarshal(msg.Payload(), &req); err != nil {
		r.publishHistoryResponse(client, topic, false, "invalid json", nil)
		return
	}
	// 优先使用 payload 中的 dev_eui，否则从 topic 提取
	if req.DevEui == "" {
		req.DevEui = devEui
	}
	if req.DevEui == "" || req.Limit <= 0 {
		r.publishHistoryResponse(client, topic, false, "invalid params", nil)
		return
	}

	var sinceTime time.Time
	if req.SinceTime != "" {
		if t, err := time.Parse(time.RFC3339, req.SinceTime); err == nil {
			sinceTime = t
		}
	}

	items, err := r.Store.GetControlHistoryByTime(req.DevEui, req.Limit, sinceTime)
	if err != nil {
		r.publishHistoryResponse(client, topic, false, err.Error(), nil)
		return
	}
	r.publishHistoryResponse(client, topic, true, "ok", items)
	log.Printf("[OK] %s queried %d control history for %s", userID, len(items), req.DevEui)
}

// handleGetSimulationHistory 解析 topic 并处理请求（同上，仅查询方法不同）
func (r *MqttClient) handleGetSimulationHistory(client mqtt.Client, msg mqtt.Message) {
	topic := msg.Topic()
	log.Println("Received GetSimulationHistory request.")
	parts := strings.Split(topic, "/")
	if len(parts) < 9 {
		r.publishHistoryResponse(client, topic, false, "invalid topic", nil)
		return
	}
	userID, _, _, devEui := parts[0], parts[3], parts[5], parts[7]

	var req HistoryRequest
	if err := json.Unmarshal(msg.Payload(), &req); err != nil {
		r.publishHistoryResponse(client, topic, false, "invalid json", nil)
		return
	}
	if req.DevEui == "" {
		req.DevEui = devEui
	}
	if req.DevEui == "" || req.Limit <= 0 {
		r.publishHistoryResponse(client, topic, false, "invalid params", nil)
		return
	}

	var sinceTime time.Time
	if req.SinceTime != "" {
		if t, err := time.Parse(time.RFC3339, req.SinceTime); err == nil {
			sinceTime = t
		}
	}

	items, err := r.Store.GetSimulationHistoryByTime(req.DevEui, req.Limit, sinceTime)
	if err != nil {
		r.publishHistoryResponse(client, topic, false, err.Error(), nil)
		return
	}
	r.publishHistoryResponse(client, topic, true, "ok", items)
	log.Printf("[OK] %s queried %d simulation history for %s", userID, len(items), req.DevEui)
}

// publishHistoryResponse 发布历史数据响应（Request→Response）
func (r *MqttClient) publishHistoryResponse(c mqtt.Client, reqTopic string, ok bool, msg string, data []store.HistoryItem) {
	respTopic := strings.Replace(reqTopic, "Request", "Response", 1)
	payload, _ := json.Marshal(map[string]interface{}{
		"success": ok,
		"message": msg,
		"data":    data,
	})
	c.Publish(respTopic, 1, false, payload)
}
