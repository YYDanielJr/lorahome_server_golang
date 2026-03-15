package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	mqttclient "lorahome_server/modules/MqttClient"
	store "lorahome_server/modules/Store"
	"net/http"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	_ "github.com/go-sql-driver/mysql"

	"database/sql"
)

type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Token   string `json:"token,omitempty"`
	Expires int64  `json:"expires,omitempty"`
}

var (
	mqttBroker = "tcp://192.168.114.121:1883"
	mqttClient mqtt.Client
	// tokenManager = TokenManager.NewTokenManager()
	mqttClientInst *mqttclient.MqttClient
	storeInstance  store.StoreIface
)

var db *sql.DB

// 响应结构体
type registerResp struct {
	UserID int `json:"userid"`
}
type queryResp struct {
	UserID int `json:"userid"`
}
type errResp struct {
	Error string `json:"error"`
}

func main() {
	var err error
	db, err = sql.Open("mysql", "yydaniel:dyy040128@tcp(127.0.0.1:3306)/lorahome?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatal("Database unreachable:", err)
	}
	gormDSN := "yydaniel:dyy040128@tcp(127.0.0.1:3306)/lorahome?charset=utf8mb4&parseTime=True&loc=Local"

	storeInstance, err = store.NewMySQLStore(gormDSN)
	if err != nil {
		log.Fatal("Store initialization failed:", err)
	}
	log.Println("✅ Store initialized (GORM managed internally)")
	mqttClientInst = mqttclient.NewMqttClient(mqttBroker)
	mqttClientInst.Store = storeInstance
	startHTTPServers()
	// 保持程序运行
	select {}
}

func startHTTPServers() {
	// POST /register
	// Body: {"name":"张三","password":"123456"}
	registerHandler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"仅支持POST"}`, http.StatusMethodNotAllowed)
			return
		}

		// 解析请求
		var req struct{ Name, Password string }
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" || req.Password == "" {
			http.Error(w, `{"error":"无效参数"}`, http.StatusBadRequest)
			return
		}

		// 1. 检查用户名是否已存在（防御重复注册）
		var cnt int
		if err := db.QueryRow("SELECT COUNT(*) FROM `UserInfo` WHERE `name`=?", req.Name).Scan(&cnt); err != nil {
			http.Error(w, `{"error":"服务异常"}`, http.StatusInternalServerError)
			return
		}
		if cnt > 0 {
			http.Error(w, `{"error":"用户名已存在"}`, http.StatusConflict)
			return
		}

		// 2. 生成userid（空表时从10001开始）
		var maxID sql.NullInt64
		db.QueryRow("SELECT MAX(`userid`) FROM `UserInfo`").Scan(&maxID)
		newID := 10001
		if maxID.Valid {
			newID = int(maxID.Int64) + 1
		}

		// 3. SHA256 哈希密码
		hash := sha256.Sum256([]byte(req.Password))
		pwdHash := hex.EncodeToString(hash[:])

		// 4. 插入数据（role=0）
		_, err := db.Exec(
			"INSERT INTO `UserInfo` (`userid`,`name`,`password_hash`,`role`) VALUES (?,?,?,?)",
			newID, req.Name, pwdHash, 0,
		)
		if err != nil {
			// 捕获唯一键冲突（如并发导致userid/name重复）
			http.Error(w, `{"error":"注册失败"}`, http.StatusInternalServerError)
			return
		}

		// 返回成功
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(registerResp{UserID: newID})
	}

	// GET /user?name=张三
	queryHandler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"仅支持GET"}`, http.StatusMethodNotAllowed)
			return
		}

		name := r.URL.Query().Get("name")
		if name == "" {
			http.Error(w, `{"error":"缺少name参数"}`, http.StatusBadRequest)
			return
		}

		// 查询userid
		var uid int
		err := db.QueryRow("SELECT `userid` FROM `UserInfo` WHERE `name`=?", name).Scan(&uid)
		if err == sql.ErrNoRows {
			http.Error(w, `{"error":"用户不存在"}`, http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, `{"error":"查询失败"}`, http.StatusInternalServerError)
			return
		}

		// 返回结果
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(queryResp{UserID: uid})
	}

	// --------------------------------------------------------------------------------------------------------
	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/get_user_id", queryHandler)
	http.HandleFunc("/deveui_map", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"仅支持GET"}`, http.StatusMethodNotAllowed)
			return
		}
		http.ServeFile(w, r, "./jsons/deveui_type_map.json")
	})
	log.Println("服务启动: http://localhost:1234")
	log.Fatal(http.ListenAndServe(":1234", nil))
}
