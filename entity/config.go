package entity

// Config 配置文件结构体
type Config struct {
	Password struct {
		Arrays []string `json:"arrays,omitempty"`
	}
}
