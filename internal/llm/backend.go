package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type TagRequest struct {
	Filename    string   `json:"filename"`
	Extension   string   `json:"extension"`
	ContentPeek string   `json:"content_peek,omitempty"`
	ImageBytes  string   `json:"image_bytes,omitempty"`
	FolderTree  []string `json:"folder_tree"`
}

type TagResponse struct {
	Tags        []string `json:"tags"`
	Destination string   `json:"destination"`
	IsNewFolder bool     `json:"is_new_folder"`
	Confidence  float64  `json:"confidence"`
	Reasoning   string   `json:"reasoning"`
}

type LLMBackend interface {
	TagContent(req TagRequest) (TagResponse, error)
}

type LMStudioBackend struct {
	Host  string
	Model string
}

func (l *LMStudioBackend) TagContent(req TagRequest) (TagResponse, error) {
	// Build prompt
	prompt := fmt.Sprintf("You are a file organiser. Given the filename %s and contents %s, and the existing folders %v, where should this file go? Return JSON.", req.Filename, req.ContentPeek, req.FolderTree)

	// Build request
	chatReq := map[string]interface{}{
		"model": l.Model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.1,
	}

	reqBytes, _ := json.Marshal(chatReq)
	httpReq, _ := http.NewRequest("POST", l.Host+"/v1/chat/completions", bytes.NewBuffer(reqBytes))
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return TagResponse{}, err
	}
	defer resp.Body.Close()

	// Parse JSON from LLM (simplified)
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	
	// Mock implementation for now
	return TagResponse{Destination: "Research/", Confidence: 0.9}, nil
}
