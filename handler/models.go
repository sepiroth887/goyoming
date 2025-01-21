package handler

import (
	"net"
	"net/http"
)

type Handler struct {
	config     Configuration
	httpClient *http.Client
}

type Event struct {
	Type   string `json:"type"`
	Length int64  `json:"data_length"`
	Data   EventData
}

type EventData map[string]any

type Client struct {
	conn net.Conn
}

type PlayMediaPayload struct {
	EntityID string `json:"entity_id"`
	Content  string `json:"media_content_id"`
	Type     string `json:"media_content_type"`
}

type TTSPayload struct {
	EntityID    string  `json:"entity_id"`
	Platform    string  `json:"tts_platform"`
	Voice       string  `json:"voice"`
	Message     string  `json:"message"`
	Announce    bool    `json:"announce"`
	VolumeLevel float64 `json:"volume_level"`
}

type Configuration struct {
	Port          int    `yaml:"port"`
	Listen        string `yaml:"listen"`
	Homeassistant struct {
		Host              string      `yaml:"host"`
		Token             interface{} `yaml:"token"`
		TargetMediaPlayer string      `yaml:"targetMediaPlayer"`
	} `yaml:"homeassistant"`
	Tts struct {
		TtsPlatform     string  `yaml:"ttsPlatform"`
		Voice           string  `yaml:"voice"`
		TtsSpeed        int     `yaml:"ttsSpeed"`
		VolumeLevel     float64 `yaml:"volumeLevel"`
		Announce        bool    `yaml:"announce"`
		MaxMessageChars int     `yaml:"maxMessageChars"`
	} `yaml:"tts"`
	ActivitySettings struct {
		RecognitionStart string `yaml:"recognitionStart"`
		RecognitionStop  string `yaml:"recognitionStop"`
		MediaFolder      string `yaml:"mediaFolder"`
	} `yaml:"activitySettings"`
}
