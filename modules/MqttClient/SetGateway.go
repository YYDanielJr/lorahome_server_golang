package mqttclient

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// SetGatewayRequest 网关设置请求体
type SetGatewayRequest struct {
	GatewayID string `json:"gateway_id"`
}

// SetGatewayResponse 网关设置响应
type SetGatewayResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// handleSetGatewayRequest 处理设置网关请求
// Topic 示例：1001/client/home/101/SetGatewayRequest
func (m *MqttClient) handleSetGatewayRequest(client mqtt.Client, msg mqtt.Message) {
	topic := msg.Topic()

	// 1. 解析 Topic: userid/client/home/homeid/SetGatewayRequest
	parts := strings.Split(topic, "/")
	// 期望长度：5 (userid, client, home, homeid, SetGatewayRequest)
	if len(parts) != 5 {
		log.Printf("Invalid topic format for SetGateway: %s", topic)
		return
	}
	userid := parts[0]
	homeidStr := parts[3]
	// homeid, _ := strconv.Atoi(homeidStr)

	// 2. 解析 Payload
	var req SetGatewayRequest
	if err := json.Unmarshal(msg.Payload(), &req); err != nil {
		log.Printf("Unmarshal request failed: %v", err)
		return
	}

	// 3. 业务逻辑校验
	code := 0
	message := "Gateway set successfully"

	// 3.1 检查网关是否存在 (根据 Gateway 表)
	exists, err := m.Store.CheckGatewayExists(req.GatewayID)
	if err != nil || !exists {
		code = 1
		message = "Gateway ID does not exist"
	} else {
		// 3.2 更新 HomeInfo 表
		// err = m.Store.UpdateHomeGateway(homeid, req.GatewayID)
		if err != nil {
			code = 1
			message = "Database update failed"
		}
	}

	// 4. 构建并发布响应
	resp := SetGatewayResponse{Code: code, Message: message}
	respJSON, _ := json.Marshal(resp)

	// Response Topic: <userid>/server/home/<homeid>/SetGatewayResponse
	respTopic := fmt.Sprintf("%s/server/home/%s/SetGatewayResponse", userid, homeidStr)

	if token := client.Publish(respTopic, 1, false, respJSON); token.Wait() && token.Error() != nil {
		log.Printf("Publish SetGateway response failed: %v", token.Error())
	}
}
