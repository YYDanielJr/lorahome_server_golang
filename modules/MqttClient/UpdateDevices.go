package mqttclient

import (
	"encoding/json"
	"fmt"
	"log"
	store "lorahome_server/modules/Store"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func (m *MqttClient) handleUpdateDevicesRequest(client mqtt.Client, msg mqtt.Message) {
	log.Println("Received an UpdateDevices Request.")
	log.Printf("[DEBUG] 原始 payload: %s", string(msg.Payload()))
	var nodes []store.NodeItem
	if json.Unmarshal(msg.Payload(), &nodes) != nil {
		return
	}

	for _, n := range nodes {
		// 直接调用 AddNode，自动处理存在/不存在逻辑
		_ = m.Store.AddNode(n.DevEUI, n.Name, n.Description, n.Type, n.JoinEUI, n.AppKey, n.GatewayID, n.RoomID)
	}
	log.Printf("已处理%d个设备", len(nodes))
}

// handleDeleteDeviceRequest 处理删除设备请求
// 流程：删除模拟历史 → 删除控制历史 → 删除设备节点
func (r *MqttClient) handleDeleteDeviceRequest(client mqtt.Client, msg mqtt.Message) {
	log.Printf("Received DeleteDevice request: %s", string(msg.Payload()))

	// 1. 解析 topic 提取参数: {userId}/client/home/{homeId}/room/{roomid}/device/{devEui}/DeleteDeviceRequest
	parts := strings.Split(msg.Topic(), "/")
	if len(parts) < 7 {
		log.Printf("Invalid DeleteDevice topic format: %s", msg.Topic())
		return
	}
	userID := parts[0]
	devEui := parts[7] // 设备唯一标识

	// 2. 可选：解析请求体（预留扩展，当前只需 devEui）
	var req map[string]interface{}
	if len(msg.Payload()) > 0 {
		_ = json.Unmarshal(msg.Payload(), &req)
	}

	log.Printf("DeleteDevice: userID=%s, devEui=%s", userID, devEui)

	// 3. 执行删除流程（按顺序：先历史，后节点）
	if err := r.Store.DeleteDeviceByDevEuiWithHistory(devEui); err != nil {
		log.Printf("DeleteDevice failed: %v", err)
		r.publishDeleteDeviceResponse(client, userID, devEui, false, err.Error())
		return
	}

	log.Printf("Deleted all data for device: %s", devEui)

	// 4. ✅ 发布成功响应
	r.publishDeleteDeviceResponse(client, userID, devEui, true, "ok")
	log.Printf("DeleteDevice success: %s", devEui)
}

// publishDeleteDeviceResponse 发布删除设备响应
func (r *MqttClient) publishDeleteDeviceResponse(client mqtt.Client, userID, devEui string, success bool, message string) {
	response := map[string]interface{}{
		"success": success,
		"message": message,
		"dev_eui": devEui,
	}
	if !success {
		response["errorCode"] = 500
	}

	payload, _ := json.Marshal(response)
	topic := fmt.Sprintf("%s/server/home/+/device/%s/DeleteDeviceResponse", userID, devEui)
	// 简化：直接返回给请求者（实际可根据 topic 动态构造）
	topic = strings.Replace(topic, "/+", "", 1) // 移除通配符，构造具体 topic

	if token := client.Publish(topic, 1, false, payload); token.Wait() && token.Error() != nil {
		log.Printf("Publish DeleteDeviceResponse failed: %v", token.Error())
	} else {
		log.Printf("Published DeleteDeviceResponse to %s", topic)
	}
}
