package config

import (
	"encoding/json"
	"os"
)

type Config struct {
    Server  ServerConfig    `json:"server"`
    Database DatabaseConfig `json:"database"`
    RabbitMQ RabbitMQConfig `json:"rabbitmq"`
    Storage  StorageConfig  `json:"storage"`
    Log      LogConfig      `json:"log"`
}

type ServerConfig struct {
	Port         int    `json:"port"`
	ReadTimeout  int    `json:"read_timeout_seconds"`
	WriteTimeout int    `json:"write_timeout_seconds"`
}

type DatabaseConfig struct {
	URL string `json:"url"`
}

type RabbitMQConfig struct {
	URL string `json:"url"`
}

type StorageConfig struct {
	UploadPath     string `json:"upload_path"`
	ProcessedPath  string `json:"processed_path"`
}

type LogConfig struct {
	Level string `json:"level"`
}

func Load(path string) (*Config, error){
    // Read the config.json file and put it in Config struct
	// so to read the file we will give the file path and return pointer Config variable
	data,err :=os.ReadFile(path)
	if err!=nil{
		return nil,err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err!=nil{
		return nil,err
	}
	return &cfg, nil	
}