package handler

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

func New(config Configuration) *Handler {
	h := &Handler{
		config:     config,
		httpClient: http.DefaultClient,
	}
	h.httpClient.Timeout = time.Second * 5
	return h
}

func (h *Handler) ListenAndServe() error {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", h.config.Listen, h.config.Port))
	if err != nil {
		return err
	}

	defer listener.Close()

	for {
		log.Info("listening for new clients")
		conn, err := listener.Accept()
		if err != nil {
			log.Errorf("failed to accept connection: %v", err)
		}

		client := Client{
			conn: conn,
		}
		go client.Handle(h)
	}
}

func (c *Client) Handle(h *Handler) {
	reader := bufio.NewReader(c.conn)
	log.Infof("new client connected: %s", c.conn.RemoteAddr().String())
	for {
		var event Event
		var eventData EventData
		m, err := reader.ReadBytes('\n')

		if err != nil {
			c.conn.Close()
			log.Errorf("failed to read from conn: %v", err)
			return
		}
		log.Debugf("received event: %s", string(m))
		err = json.Unmarshal(m, &event)
		if err != nil {
			log.Errorf("failed to decode event: %s\n%v", string(m), err)
			continue
		}

		if event.Length <= 0 {
			// log.Println("received no addl data in event.")
			continue
		}
		log.Debugf("reading %d bytes", event.Length)
		dataBuf := make([]byte, event.Length)

		n, err := io.ReadFull(reader, dataBuf)
		if err != nil {
			log.Errorf("failed to read event data: %v\n", err)
			continue
		}

		log.Debugf("read %d bytes", n)

		err = json.Unmarshal(dataBuf, &eventData)
		if err != nil {
			fmt.Printf("failed to decode event data: %v\n", err)
			continue
		}
		event.Data = eventData
		log.Debug(event)

		switch event.Type {
		case "detection":
			go h.playSound(h.config.ActivitySettings.RecognitionStart)
		case "voice-stopped":
			go h.playSound(h.config.ActivitySettings.RecognitionStop)
		case "synthesize":
			go h.syntesize(event.Data["text"].(string))
		}
	}
}

func (h *Handler) playSound(mediaFile string) {
	source := fmt.Sprintf("media-source://media_source/local/%s/%s", h.config.ActivitySettings.MediaFolder, mediaFile)
	payload := PlayMediaPayload{
		EntityID: h.config.Homeassistant.TargetMediaPlayer,
		Content:  source,
		Type:     "audio/mpeg",
	}
	data, err := json.Marshal(&payload)
	if err != nil {
		log.Errorf("failed to create voice notification payload: %v", err)
		return
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/services/media_player/play_media", h.config.Homeassistant.Host), bytes.NewBuffer(data))
	if err != nil {
		log.Errorf("failed to create voice notification request: %v", err)
		return
	}

	req.Header.Set("authorization", fmt.Sprintf("Bearer %s", h.config.Homeassistant.Token))
	req.Header.Set("content-type", "application/json")

	res, err := h.httpClient.Do(req)
	if err != nil || res.StatusCode < 200 || res.StatusCode > 299 {
		log.Errorf("failed to make voice notification request [%d]: %v", res.StatusCode, err)
	}
}

func (h *Handler) syntesize(rawText string) {

	text := strings.NewReplacer("*", "", "&", "", "[", "", "]", "", "(", "", ")", "", "{", "", "}", "").Replace(rawText)

	payload := TTSPayload{
		EntityID:    h.config.Homeassistant.TargetMediaPlayer,
		Platform:    h.config.Tts.TtsPlatform,
		Voice:       h.config.Tts.Voice,
		Announce:    h.config.Tts.Announce,
		VolumeLevel: h.config.Tts.VolumeLevel,
		Message:     text,
	}

	data, err := json.Marshal(&payload)
	if err != nil {
		log.Errorf("failed to create tts notification payload: %v", err)
		return
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/services/chime_tts/say", h.config.Homeassistant.Host), bytes.NewBuffer(data))
	if err != nil {
		log.Errorf("failed to create tts notification request: %v", err)
		return
	}

	req.Header.Set("authorization", fmt.Sprintf("Bearer %s", h.config.Homeassistant.Token))
	req.Header.Set("content-type", "application/json")

	h.httpClient.Timeout = 10 * time.Second
	defer func() {
		h.httpClient.Timeout = 5 * time.Second
	}()
	res, err := h.httpClient.Do(req)

	if err != nil {
		log.Errorf("failed to make tts notification request: %v", err)
		return
	}

	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		data, _ := io.ReadAll(res.Body)
		log.Errorf("unexpected response from HA %d: %s", res.StatusCode, string(data))
	}
}
