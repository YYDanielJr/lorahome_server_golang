package mqttclient

import (
	"encoding/json"
	"log"
	store "lorahome_server/modules/Store"

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
