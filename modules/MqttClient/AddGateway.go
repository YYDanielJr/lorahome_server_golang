package mqttclient

import (
	"log"
)

func (m *MqttClient) AddGateway(gatewayid string) int {
	isSuccess, err := m.Store.AddGateway(gatewayid)
	if isSuccess {
		return 0
	} else {
		log.Println("Adding gateway error: " + err.Error())
		return 1
	}
}
