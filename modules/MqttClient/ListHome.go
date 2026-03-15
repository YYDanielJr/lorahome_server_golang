package mqttclient

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// ListHomeResponse 返回给客户端的家庭列表
type HomeItem struct {
	HomeID    int     `json:"homeid"`
	Name      string  `json:"name"`
	Latitude  float32 `json:"location_latitude,omitempty"`
	Longitude float32 `json:"location_longitude,omitempty"`
	GatewayID string  `json:"gateway_id,omitempty"`
}

type ListHomeResponse struct {
	Homes []HomeItem `json:"homes"`
}

type GetHomeByGatewayIdResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	HomeID  string `json:"home_id,omitempty"`
}

// handleListHomeRequest 处理获取家庭列表请求
// Topic 示例：1001/client/ListHomeRequest
func (m *MqttClient) handleListHomeRequest(client mqtt.Client, msg mqtt.Message) {
	log.Printf("Received a ListHome Request.")
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

	// 调用 Store 接口查询
	storeHomes, err := m.Store.GetHomesByUserID(userid)
	if err != nil {
		log.Printf("Get homes failed: %v", err)
		return
	}

	homes := make([]HomeItem, 0, len(storeHomes))
	for _, h := range storeHomes {
		homes = append(homes, HomeItem{
			HomeID:    h.HomeID,
			Name:      h.Name,
			Latitude:  h.Latitude,
			Longitude: h.Longitude,
			GatewayID: h.GatewayID,
		})
	}

	resp := ListHomeResponse{Homes: homes}
	respJSON, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Marshal response failed: %v", err)
		return
	}

	respTopic := fmt.Sprintf("%s/server/ListHomeResponse", useridStr)
	if token := client.Publish(respTopic, 1, false, respJSON); token.Wait() && token.Error() != nil {
		log.Printf("Publish response failed: %v", token.Error())
	}
}

// // handleGetHomeByGatewayIdRequest 根据网关ID查询家庭
// // Topic: <userid>/client/home/GetHomeByGatewayIdRequest
// func (m *MqttClient) handleGetHomeByGatewayIdRequest(client mqtt.Client, msg mqtt.Message) {
// 	topic := msg.Topic()
// 	log.Println("Received GetHomeByGatewayIdRequest")
// 	parts := strings.Split(topic, "/")
// 	if len(parts) < 3 {
// 		return
// 	}

// 	userid := parts[0]

// 	// 解析 payload 获取 gateway_id
// 	var req struct {
// 		GatewayID string `json:"gateway_id"`
// 	}
// 	if json.Unmarshal(msg.Payload(), &req) != nil || req.GatewayID == "" {
// 		return
// 	}

// 	// 查询数据库（返回 *store.HomeItem）
// 	log.Println("Received GatewayID: " + req.GatewayID)
// 	storeHome, err := m.Store.GetHomeByGatewayId(req.GatewayID)

// 	// 类型转换：store.HomeItem → 本地 HomeItem
// 	var home *HomeItem
// 	if err == nil && storeHome != nil {
// 		home = &HomeItem{
// 			HomeID:    storeHome.HomeID,
// 			Name:      storeHome.Name,
// 			Latitude:  storeHome.Latitude,
// 			Longitude: storeHome.Longitude,
// 			GatewayID: storeHome.GatewayID,
// 		}
// 	}
// 	if err != nil {
// 		log.Fatalln(err)
// 	}
// 	if storeHome == nil {
// 		log.Fatalln("Store Home not exist.")
// 	}

// 	// 构造并发送响应
// 	resp := GetHomeByGatewayIdResponse{
// 		Success: err == nil,
// 		Message: func() string {
// 			if err != nil {
// 				return err.Error()
// 			}
// 			return "ok"
// 		}(),
// 		Home: func() *HomeItem {
// 			if err == nil {
// 				return home
// 			}
// 			return nil
// 		}(),
// 	}

//		payload, _ := json.Marshal(resp)
//		client.Publish(fmt.Sprintf("%s/server/home/GetHomeByGatewayIdResponse", userid), 1, false, payload)
//		log.Println(resp)
//		log.Println("Published to ", fmt.Sprintf("%s/server/home/GetHomeByGatewayIdResponse", userid))
//	}
//
// handleGetHomeByGatewayIdRequest 根据网关ID查询家庭
func (m *MqttClient) handleGetHomeByGatewayIdRequest(client mqtt.Client, msg mqtt.Message) {
	topic := msg.Topic()
	parts := strings.Split(topic, "/")
	if len(parts) < 3 {
		return
	}

	userid := parts[0]

	var req struct {
		GatewayID string `json:"gateway_id"`
	}
	if json.Unmarshal(msg.Payload(), &req) != nil || req.GatewayID == "" {
		return
	}

	// 查询数据库
	storeHome, err := m.Store.GetHomeByGatewayId(req.GatewayID)

	// ✅ 构造响应：只返回 HomeID 字符串
	resp := GetHomeByGatewayIdResponse{
		Success: err == nil && storeHome != nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			if storeHome == nil {
				return "not found"
			}
			return "ok"
		}(),
		HomeID: func() string {
			if err == nil && storeHome != nil {
				return fmt.Sprintf("%d", storeHome.HomeID) // ✅ 转为字符串
			}
			return ""
		}(),
	}

	payload, _ := json.Marshal(resp)
	client.Publish(fmt.Sprintf("%s/server/home/GetHomeByGatewayIdResponse", userid), 1, false, payload)
	log.Printf("[INFO] 响应: %s", string(payload)) // ✅ 打印原始JSON便于调试
}
