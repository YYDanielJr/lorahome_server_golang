package mqttclient

import (
	"encoding/json"
	"log"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// handleDataUpload 处理设备数据上传消息
// 主题格式: {userid}/gateway/home/{homeid}/room/1/device/{dev_eui}/DataUpload
// 数据格式:
//   - 温湿度: {"humidity":46,"temperature":21}
//   - 灯泡:   {"is_open":1} 或 {"is_open":0}
func (r *MqttClient) handleDataUpload(client mqtt.Client, msg mqtt.Message) {
	topic := msg.Topic()
	payload := msg.Payload()

	// 1. 解析 topic 提取 userid 和 dev_eui
	// 格式: userid/gateway/home/homeid/room/1/device/dev_eui/DataUpload
	parts := strings.Split(topic, "/")
	if len(parts) < 9 {
		log.Printf("[WARN] Invalid upload topic: %s", topic)
		return
	}
	// userID := parts[0]
	devEui := parts[7] // device 后的第 2 个元素

	// 2. 解析数据载荷
	// 先尝试解析为通用 map
	var data map[string]interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		log.Printf("[ERROR] Failed to unmarshal payload: %s | %v", string(payload), err)
		return
	}

	// 3. 判断数据类型并记录
	// 情况 1: 温湿度数据 (含 temperature 或 humidity 字段)
	if temp, ok := data["temperature"].(float64); ok {
		if err := r.Store.RecordSimulationData(devEui, 1, temp); err != nil {
			log.Printf("[ERROR] Record temperature failed: %v", err)
		} else {
			log.Printf("[OK] Recorded temp=%.2f for %s", temp, devEui)
		}
	}
	if humi, ok := data["humidity"].(float64); ok {
		if err := r.Store.RecordSimulationData(devEui, 2, humi); err != nil {
			log.Printf("[ERROR] Record humidity failed: %v", err)
		} else {
			log.Printf("[OK] Recorded humi=%.2f for %s", humi, devEui)
		}
	}

	// 情况 2: 灯泡控制数据 (含 is_open 字段)
	if isOpenVal, ok := data["is_open"]; ok {
		var isOpen bool
		switch v := isOpenVal.(type) {
		case float64: // JSON 数字默认解析为 float64
			isOpen = v == 1
		case string:
			isOpen = v == "1"
		case bool:
			isOpen = v
		default:
			log.Printf("[WARN] Unknown is_open type: %T", isOpenVal)
			return
		}
		if err := r.Store.RecordControlData(devEui, isOpen); err != nil {
			log.Printf("[ERROR] Record control failed: %v", err)
		} else {
			log.Printf("[OK] Recorded is_open=%v for %s", isOpen, devEui)
		}
	}
}

// SubscribeDataUploadTopics 订阅所有设备的数据上传主题
// 通配符: +/gateway/home/+/room/1/device/+/DataUpload
// func (r *MqttClient) SubscribeDataUploadTopics() {
// 	topic := "+/gateway/home/+/room/1/device/+/DataUpload"
// 	if token := r.Client.Subscribe(topic, 1, r.handleDataUpload); token.Wait() && token.Error() != nil {
// 		log.Printf("[ERROR] Subscribe DataUpload failed: %v", token.Error())
// 	} else {
// 		log.Printf("[INFO] Subscribed topic: %s", topic)
// 	}
// }
