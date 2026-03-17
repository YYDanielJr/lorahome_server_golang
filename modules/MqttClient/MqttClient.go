package mqttclient

/**
* mqtt交流协议：
* 1. 分为三个部分，分别是gateway、client和server。消息从哪里发送，第一级主题就是谁。
* 	例如，消息来自网关，那么client需要接收的地址就是<userid>/gateway/+。
 */

/**
* 服务器端mqtt client的作用：
* 1. 向安卓客户端返回家庭列表，request主题地址：<userid>/client/ListHomeRequest
* 	response主题地址：<userid>/server/ListHomeResponse
* 2. 为没有设置Gateway的Home设置gatewayid，request主题地址：<userid>/client/home/<homeid>/SetGatewayRequest
* 	response主题地址：<userid>/server/home/<homeid>/SetGatewayResponse
 */

/**
* 服务器登录emqx账户：
* ClientID: LoraHome_golang_server
* Username: -1
* Password: fY6yS3lR4lJ8rV7r
 */

import (
	"log"
	"time"

	store "lorahome_server/modules/Store"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	_ "github.com/go-sql-driver/mysql" // MySQL 驱动
)

// ----------------------------------------------------------------------------------------

type MqttClient struct {
	Client mqtt.Client
	Broker string
	Store  store.StoreIface
}

func NewMqttClient(mqttBroker string) *MqttClient {

	log.Printf("Creating Mqtt Client...")

	r := &MqttClient{
		Broker: mqttBroker,
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(mqttBroker)
	opts.SetClientID("LoraHome_golang_server")
	opts.SetUsername("-1")
	opts.SetPassword("fY6yS3lR4lJ8rV7r")
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)

	opts.SetOnConnectHandler(func(c mqtt.Client) {
		log.Println("MQTT Connected")
		// 订阅获取用户信息请求：例如 1001/client/GetUserInfoRequest
		if token := c.Subscribe("+/client/GetUserInfoRequest", 1, func(client mqtt.Client, msg mqtt.Message) {
			r.handleGetUserInfoRequest(client, msg)
		}); token.Wait() && token.Error() != nil {
			log.Printf("Subscribe GetUserInfoRequest failed: %v", token.Error())
		} else {
			log.Printf("Subscribed GetUserInfoRequest successfully.")
		}

		// 订阅家庭列表请求：例如 1001/client/ListHomeRequest
		if token := c.Subscribe("+/client/ListHomeRequest", 1, func(client mqtt.Client, msg mqtt.Message) {
			r.handleListHomeRequest(client, msg)
		}); token.Wait() && token.Error() != nil {
			log.Printf("Subscribe ListHomeRequest failed: %v", token.Error())
		} else {
			log.Printf("Subscribed ListHomeRequest successfully.")
		}

		// 订阅根据网关查家庭请求
		if token := c.Subscribe("+/client/home/GetHomeByGatewayIdRequest", 1, func(client mqtt.Client, msg mqtt.Message) {
			r.handleGetHomeByGatewayIdRequest(client, msg)
		}); token.Wait() && token.Error() != nil {
			log.Printf("Subscribe GetHomeByGatewayIdRequest failed: %v", token.Error())
		} else {
			log.Printf("Subscribed GetHomeByGatewayIdRequest successfully.")
		}

		// 订阅设置网关请求：例如 1001/client/home/101/SetGatewayRequest
		if token := c.Subscribe("+/client/home/+/SetGatewayRequest", 1, func(client mqtt.Client, msg mqtt.Message) {
			r.handleSetGatewayRequest(client, msg)
		}); token.Wait() && token.Error() != nil {
			log.Printf("Subscribe SetGatewayRequest failed: %v", token.Error())
		} else {
			log.Printf("Subscribed SetGatewayRequest successfully.")
		}

		// 订阅新建家庭请求：例如 1001/client/AddHomeRequest
		if token := c.Subscribe("+/client/AddHomeRequest", 1, func(client mqtt.Client, msg mqtt.Message) {
			r.handleAddHomeRequest(client, msg)
		}); token.Wait() && token.Error() != nil {
			log.Printf("Subscribe AddHomeRequest failed: %v", token.Error())
		} else {
			log.Printf("Subscribed AddHomeRequest successfully.")
		}

		// 订阅更新节点列表请求: <userid>/gateway/home/<homeid>/room/<roomid>/UpdateDevices
		if token := c.Subscribe("+/gateway/home/+/room/+/UpdateDevices", 1, func(client mqtt.Client, msg mqtt.Message) {
			r.handleUpdateDevicesRequest(client, msg)
		}); token.Wait() && token.Error() != nil {
			log.Printf("Subscribe UpdateDevices failed: %v", token.Error())
		} else {
			log.Printf("Subscribed UpdateDevices successfully.")
		}

		// 订阅删除设备请求：例如 1001/client/home/101/room/xxx/device/yyy/DeleteDevice
		if token := c.Subscribe("+/+/home/+/room/+/device/+/DeleteDevice", 1, func(client mqtt.Client, msg mqtt.Message) {
			r.handleDeleteDeviceRequest(client, msg)
		}); token.Wait() && token.Error() != nil {
			log.Printf("Subscribe DeleteDeviceRequest failed: %v", token.Error())
		} else {
			log.Printf("Subscribed DeleteDeviceRequest successfully.")
		}

		// 订阅设备数据上传主题
		// 格式: {userid}/gateway/home/{homeid}/room/1/device/{dev_eui}/DataUpload
		if token := c.Subscribe("+/gateway/home/+/room/+/device/+/DataUpload", 1, func(client mqtt.Client, msg mqtt.Message) {
			r.handleDataUpload(client, msg)
		}); token.Wait() && token.Error() != nil {
			log.Printf("Subscribe DataUpload failed: %v", token.Error())
		} else {
			log.Printf("Subscribed DataUpload successfully.")
		}

		// 订阅控制历史查询请求
		if token := c.Subscribe("+/+/home/+/room/+/device/+/GetControlHistoryRequest", 1, func(client mqtt.Client, msg mqtt.Message) {
			r.handleGetControlHistory(client, msg)
		}); token.Wait() && token.Error() != nil {
			log.Printf("Subscribe GetControlHistoryRequest failed: %v", token.Error())
		} else {
			log.Printf("Subscribed GetControlHistoryRequest successfully.")
		}

		// 订阅模拟历史查询请求
		if token := c.Subscribe("+/+/home/+/room/+/device/+/GetSimulationHistoryRequest", 1, func(client mqtt.Client, msg mqtt.Message) {
			r.handleGetSimulationHistory(client, msg)
		}); token.Wait() && token.Error() != nil {
			log.Printf("Subscribe GetSimulationHistoryRequest failed: %v", token.Error())
		} else {
			log.Printf("Subscribed GetSimulationHistoryRequest successfully.")
		}
	})

	opts.SetConnectionLostHandler(func(c mqtt.Client, err error) {
		log.Printf("Connection lost: %v", err)
	})

	mqttClient := mqtt.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", token.Error())
	}
	r.Client = mqttClient
	return r
}

// --------------------------------------------------------------------------------------------------------------------
