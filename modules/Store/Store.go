package store

import (
	"errors"
	"fmt"
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

// StoreIface 接口定义（供 mqttclient 依赖）
type StoreIface interface {
	GetHomesByUserID(userid int) ([]HomeItem, error)
	GetHomeByGatewayId(gatewayId string) (*HomeItem, error)
	UpdateHomeGateway(homeid int, gatewayID string) error
	CheckGatewayExists(gatewayid string) (bool, error)
	AddHome(userid int, homename string, gatewayid string) (*HomeItem, error)
	AddGateway(gatewayid string) (bool, error)
	AddRoom(name string, homeid int) (*RoomItem, error)
	AddNode(devEUI, name, description string, typ int, joinEUI, appKey, gatewayID string, roomID int) error
	RecordSimulationData(devEui string, dataType int, value float64) error
	RecordControlData(devEui string, isOpen bool) error
	GetControlHistory(devEui string, limit int) ([]HistoryItem, error)
	GetSimulationHistory(devEui string, limit int) ([]HistoryItem, error)
	GetControlHistoryByTime(devEui string, limit int, sinceTime time.Time) ([]HistoryItem, error)
	GetSimulationHistoryByTime(devEui string, limit int, sinceTime time.Time) ([]HistoryItem, error)
	GetDeviceHistory(devEui string, limit int) ([]HistoryItem, error)
}

// 复用 mqttclient 中的数据结构（或在此重新定义）
type HomeItem struct {
	HomeID    int     `json:"homeid" gorm:"column:homeid"`
	Name      string  `json:"name" gorm:"column:name"`
	Latitude  float32 `json:"location_latitude,omitempty" gorm:"column:location_latitude"`
	Longitude float32 `json:"location_longitude,omitempty" gorm:"column:location_longitude"`
	UserID    int     `json:"userid" gorm:"column:userid"`
	GatewayID string  `json:"gateway_id,omitempty" gorm:"column:gateway_id"`
}

type Gateway struct {
	GatewayID string     `gorm:"column:gateway_id;primaryKey"`
	LastSeen  *time.Time `gorm:"column:last_seen"`
	IsOnline  bool       `gorm:"column:is_online"`
}

type RoomItem struct {
	RoomID int    `json:"room_id" gorm:"column:room_id;primaryKey"`
	Name   string `json:"name" gorm:"column:name"`
	HomeID int    `json:"homeid" gorm:"column:homeid"`
}

type NodeItem struct {
	DevEUI      string     `json:"dev_eui" gorm:"column:dev_eui;primaryKey"`
	Name        string     `json:"name" gorm:"column:name"`
	Type        int        `json:"type" gorm:"column:type"`
	JoinEUI     string     `json:"join_eui" gorm:"column:join_eui"`
	AppKey      string     `json:"app_key" gorm:"column:app_key"`
	CreatedAt   time.Time  `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	IsOnline    bool       `json:"is_online" gorm:"column:is_online"`
	LastSeen    *time.Time `json:"last_seen" gorm:"column:last_seen"`
	GatewayID   string     `json:"gateway_id" gorm:"column:gateway_id"`
	RoomID      int        `json:"room_id" gorm:"column:room_id"`
	Description string     `json:"description,omitempty" gorm:"column:description"`
}

type SimulationHistroyData struct {
	ID          uint      `gorm:"column:id;primaryKey;autoIncrement"`
	ReceiveTime time.Time `gorm:"column:receive_time;autoCreateTime"`
	Type        int       `gorm:"column:type"`          // 1=温度, 2=湿度
	Value       float64   `gorm:"column:value"`         // 温湿度数值
	DevEui      string    `gorm:"column:dev_eui;index"` // 外键索引
}

type ControlHistoryData struct {
	ID          uint      `gorm:"column:id;primaryKey;autoIncrement"`
	ReceiveTime time.Time `gorm:"column:receive_time;autoCreateTime"`
	Type        int       `gorm:"column:type"`    // -1=灯控制
	IsOpen      string    `gorm:"column:is_open"` // "1"=开, "0"=关
	DevEui      string    `gorm:"column:dev_eui;index"`
}

type HistoryItem struct {
	ReceiveTime time.Time `json:"receive_time"`
	Type        int       `json:"type"`              // 1=温度,2=湿度,-1=灯控制
	Value       float64   `json:"value,omitempty"`   // 温湿度值（控制数据时为0）
	IsOpen      string    `json:"is_open,omitempty"` // "1"/"0"（仅控制数据）
}

type MySQLStore struct {
	db *gorm.DB
}

// NewMySQLStore 初始化数据库连接
func NewMySQLStore(dsn string) (*MySQLStore, error) {
	// 配置 GORM 日志级别（生产环境建议 Warn 或 Error）
	gormLogger := logger.New(
		log.New(log.Writer(), "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond, // 慢查询阈值
			LogLevel:                  logger.Warn,            // 日志级别
			IgnoreRecordNotFoundError: true,                   // 忽略 ErrRecordNotFound
			Colorful:                  true,                   // 彩色输出
		},
	)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("gorm.Open failed: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("db.DB() failed: %w", err)
	}

	// 配置连接池（根据实际需求调整）
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	log.Println("[Store] MySQL connected successfully")
	return &MySQLStore{db: db}, nil
}

// GetHomesByUserID 查询用户的所有家庭
func (s *MySQLStore) GetHomesByUserID(userid int) ([]HomeItem, error) {
	var homes []HomeItem
	err := s.db.Table("HomeInfo").
		Select("homeid, name, location_latitude, location_longitude, gateway_id").
		Where("userid = ?", userid).
		Find(&homes).Error
	return homes, err
}

// GetHomeByGatewayId 根据网关ID查询家庭
func (s *MySQLStore) GetHomeByGatewayId(gatewayId string) (*HomeItem, error) {
	var home HomeItem
	err := s.db.Table("HomeInfo").
		Where("gateway_id = ?", gatewayId).
		First(&home).Error
	return &home, err
}

// UpdateHomeGateway 更新家庭的 gateway_id
func (s *MySQLStore) UpdateHomeGateway(homeid int, gatewayID string) error {
	return s.db.Table("HomeInfo").
		Where("homeid = ?", homeid).
		Update("gateway_id", gatewayID).Error
}

// CheckGatewayExists 检查网关是否存在
func (s *MySQLStore) CheckGatewayExists(gatewayid string) (bool, error) {
	var count int64
	err := s.db.Table("Gateway").
		Where("gateway_id = ?", gatewayid).
		Count(&count).Error
	return count > 0, err
}

// // GetLastHomeId 获取当前最大的HomeID
// func (s *MySQLStore) GetLastHomeId() (int, error) {
// 	var id int
// 	err := s.db.Table("HomeInfo").Select()
// }

// AddHome 添加家庭
func (s *MySQLStore) AddHome(userid int, homename string, gatewayid string) (*HomeItem, error) {
	// 检查是否已存在
	var existingHome HomeItem
	err := s.db.Table("HomeInfo").
		Where("userid = ? AND name = ?", userid, homename).
		First(&existingHome).Error

	if err == nil {
		return nil, fmt.Errorf("家庭名称已存在")
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// 创建新家庭（包含 userid）
	newHome := HomeItem{
		Name:      homename,
		UserID:    userid,
		GatewayID: gatewayid,
	}
	err = s.db.Table("HomeInfo").Create(&newHome).Error
	if err != nil {
		return nil, err
	}

	return &newHome, nil
}

// 添加网关
func (s *MySQLStore) AddGateway(gatewayID string) (bool, error) {
	now := time.Now()
	gateway := Gateway{
		GatewayID: gatewayID,
		LastSeen:  &now,
		IsOnline:  false,
	}

	result := s.db.Table("Gateway").Create(&gateway)
	if result.Error != nil {
		return false, result.Error
	}
	// 插入成功
	return true, nil
}

// AddRoom 添加房间
func (s *MySQLStore) AddRoom(name string, homeid int) (*RoomItem, error) {
	room := RoomItem{Name: name, HomeID: homeid}
	err := s.db.Table("RoomInfo").Create(&room).Error
	return &room, err
}

// AddNode 添加/更新设备节点
func (s *MySQLStore) AddNode(devEUI, name, description string, typ int, joinEUI, appKey, gatewayID string, roomID int) error {
	node := NodeItem{
		DevEUI: devEUI, Name: name, Description: description, Type: typ,
		JoinEUI: joinEUI, AppKey: appKey,
		GatewayID: gatewayID, RoomID: roomID,
	}
	return s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "dev_eui"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "type", "join_eui", "app_key", "gateway_id", "room_id", "is_online", "last_seen", "description"}), // ✅ 新增 description
	}).Table("NodeInfo").Create(&node).Error
}

// RecordSimulationData 记录模拟数据（温度 type=1, 湿度 type=2）
// 存入表: SimulationHistroyData
func (s *MySQLStore) RecordSimulationData(devEui string, dataType int, value float64) error {
	return s.db.Table("SimulationHistroyData").Create(&SimulationHistroyData{
		Type:   dataType,
		Value:  value,
		DevEui: devEui,
	}).Error
}

func (s *MySQLStore) RecordControlData(devEui string, isOpen bool) error {
	openVal := "0"
	if isOpen {
		openVal = "1"
	}
	return s.db.Table("ControlHistoryData").Create(&ControlHistoryData{
		Type:   -1,
		IsOpen: openVal,
		DevEui: devEui,
	}).Error
}

// GetControlHistory 获取指定设备的灯控制历史（type=-1），按时间倒序
func (s *MySQLStore) GetControlHistory(devEui string, limit int) ([]HistoryItem, error) {
	var items []HistoryItem
	err := s.db.Table("ControlHistoryData").
		Select("receive_time, type, is_open").
		Where("dev_eui = ? AND type = -1", devEui).
		Order("receive_time DESC").
		Limit(limit).
		Scan(&items).Error
	return items, err
}

// GetSimulationHistory 获取指定设备的温湿度历史（type=1/2），按时间倒序
func (s *MySQLStore) GetSimulationHistory(devEui string, limit int) ([]HistoryItem, error) {
	var items []HistoryItem
	err := s.db.Table("SimulationHistroyData").
		Select("receive_time, type, value").
		Where("dev_eui = ? AND type IN (1, 2)", devEui).
		Order("receive_time DESC").
		Limit(limit).
		Scan(&items).Error
	return items, err
}

// GetDeviceHistory 通用方法：获取指定设备的所有历史（控制+模拟），按时间倒序
func (s *MySQLStore) GetDeviceHistory(devEui string, limit int) ([]HistoryItem, error) {
	var items []HistoryItem
	// 联合查询两张表（MySQL 8.0+ 支持 UNION）
	err := s.db.Raw(`
		(SELECT receive_time, type, value, '' AS is_open FROM SimulationHistroyData WHERE dev_eui = ? AND type IN (1,2))
		UNION ALL
		(SELECT receive_time, type, 0 AS value, is_open FROM ControlHistoryData WHERE dev_eui = ? AND type = -1)
		ORDER BY receive_time DESC LIMIT ?
	`, devEui, devEui, limit).Scan(&items).Error
	return items, err
}

// GetControlHistory 获取控制历史（type=-1）
// sinceTime 为零值时查询全部，否则查询该时间之后的数据
func (s *MySQLStore) GetControlHistoryByTime(devEui string, limit int, sinceTime time.Time) ([]HistoryItem, error) {
	var items []HistoryItem
	query := s.db.Table("ControlHistoryData").
		Select("receive_time, type, is_open").
		Where("dev_eui = ? AND type = -1", devEui)

	if !sinceTime.IsZero() {
		query = query.Where("receive_time >= ?", sinceTime)
	}

	err := query.Order("receive_time DESC").Limit(limit).Scan(&items).Error
	return items, err
}

// GetSimulationHistory 获取模拟数据历史（type=1/2）
// sinceTime 为零值时查询全部，否则查询该时间之后的数据
func (s *MySQLStore) GetSimulationHistoryByTime(devEui string, limit int, sinceTime time.Time) ([]HistoryItem, error) {
	var items []HistoryItem
	query := s.db.Table("SimulationHistroyData").
		Select("receive_time, type, value").
		Where("dev_eui = ? AND type IN (1, 2)", devEui)

	if !sinceTime.IsZero() {
		query = query.Where("receive_time >= ?", sinceTime)
	}

	err := query.Order("receive_time DESC").Limit(limit).Scan(&items).Error
	return items, err
}
