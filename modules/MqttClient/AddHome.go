package mqttclient

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func (m *MqttClient) handleAddHomeRequest(client mqtt.Client, msg mqtt.Message) {
	log.Printf("Received a AddHome Request.")
	log.Println(string(msg.Payload()))
	topic := msg.Topic()

	parts := strings.Split(topic, "/")
	if len(parts) < 3 {
		log.Printf("Invalid topic format: %s", topic)
		return
	}
	useridStr := parts[0]
	userid, err := strconv.Atoi(useridStr)
	if err != nil {
		log.Printf("Invalid userid in topic: %s", useridStr)
		return
	}

	var payloadData struct {
		Name       string `json:"name"`
		Gateway_Id string `json:"gateway_id"`
	}

	err = json.Unmarshal(msg.Payload(), &payloadData)
	if err != nil {
		log.Printf("解析 JSON payload 失败：%v", err)
		// 发送错误响应
		m.sendAddHomeResponse(userid, false, "JSON 解析失败", 0)
		return
	}
	gatewayid := strings.TrimSpace(payloadData.Gateway_Id)
	if gatewayid != "" {
		m.AddGateway(gatewayid)
	}
	homename := strings.TrimSpace(payloadData.Name)
	if homename == "" {
		log.Printf("家庭名称不能为空")
		m.sendAddHomeResponse(userid, false, "家庭名称不能为空", 0)
		return
	}

	homeItem, err := m.Store.AddHome(userid, homename, gatewayid)
	if err != nil {
		log.Printf("添加家庭失败：%v", err)
		// 发送错误响应
		m.sendAddHomeResponse(userid, false, err.Error(), 0)
		return
	}

	_, _ = m.Store.AddRoom("default", homeItem.HomeID)

	log.Printf("成功添加家庭：homeid=%d, name=%s, userid=%d",
		homeItem.HomeID, homeItem.Name, userid)
	m.sendAddHomeResponse(userid, true, "添加成功", homeItem.HomeID)

	log.Printf("成功添加家庭：homeid=%d, name=%s, userid=%d",
		homeItem.HomeID, homeItem.Name, userid)
	m.sendAddHomeResponse(userid, true, "添加成功", homeItem.HomeID)
}

func (m *MqttClient) sendAddHomeResponse(userid int, success bool, message string, homeid int) {
	responseTopic := fmt.Sprintf("%d/server/AddHomeResponse", userid)

	// 构建响应 JSON
	responseData := map[string]interface{}{
		"success": success,
		"message": message,
		"homeid":  homeid,
	}

	responsePayload, err := json.Marshal(responseData)
	if err != nil {
		log.Printf("构建响应 JSON 失败：%v", err)
		return
	}

	// 发布响应消息
	token := m.Client.Publish(responseTopic, 1, false, responsePayload)
	token.Wait()
	if token.Error() != nil {
		log.Printf("发送响应失败：%v", token.Error())
	}
}
