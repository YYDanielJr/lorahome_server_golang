package mqttclient

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	store "lorahome_server/modules/Store"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// handleGetUserInfoRequest 处理用户信息请求
func (r *MqttClient) handleGetUserInfoRequest(client mqtt.Client, msg mqtt.Message) {
	// 解析 topic: {userid}/client/GetUserInfoRequest
	parts := strings.Split(msg.Topic(), "/")
	if len(parts) < 3 {
		log.Printf("Invalid topic format: %s", msg.Topic())
		return
	}

	userID := parts[0]
	log.Printf("GetUserInfoRequest from user: %s", userID)

	// 查询用户信息
	uid, err := strconv.Atoi(userID)
	if err != nil {
		log.Printf("Invalid userID format: %s", userID)
		r.sendUserInfoResponse(client, userID, nil, "参数错误")
		return
	}

	info, err := r.Store.GetUserInfo(uid)
	if err != nil {
		log.Printf("GetUserInfo failed for user %s: %v", userID, err)
		r.sendUserInfoResponse(client, userID, nil, "用户不存在")
		return
	}

	// 返回成功响应
	r.sendUserInfoResponse(client, userID, info, "")
}

// sendUserInfoResponse 发送用户信息响应
func (r *MqttClient) sendUserInfoResponse(client mqtt.Client, userID string, info *store.UserInfo, errMsg string) {
	response := map[string]interface{}{
		"success": errMsg == "",
	}
	if errMsg != "" {
		response["message"] = errMsg
	} else if info != nil {
		response["userid"] = info.UserID
		response["username"] = info.Username
		response["avatar"] = info.Avatar
	}

	payload, _ := json.Marshal(response)
	topic := fmt.Sprintf("%s/server/GetUserInfoResponse", userID)

	if token := client.Publish(topic, 1, false, payload); token.Wait() && token.Error() != nil {
		log.Printf("Publish GetUserInfoResponse failed: %v", token.Error())
	} else {
		log.Printf("Sent GetUserInfoResponse to %s", topic)
	}
}
