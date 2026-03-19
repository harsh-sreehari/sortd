package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type TagResponse struct {
	Tags        []string `json:"tags"`
	Destination string   `json:"destination"`
	IsNewFolder bool     `json:"is_new_folder"`
	Confidence  float64  `json:"confidence"`
	Reasoning   string   `json:"reasoning"`
}

type TagRequest struct {
	Filename     string   `json:"filename"`
	Extension    string   `json:"extension"`
	ContentPeek  string   `json:"content_peek,omitempty"`
	FolderTree   []string `json:"folder_tree"`
	AllowedRoots []string `json:"allowed_roots"`
}

type LLMBackend interface {
	TagContent(req TagRequest) (TagResponse, error)
	ResolveReview(userInput string, filename string, tree []string) (TagResponse, error)
}

type LMStudioBackend struct {
	Host  string
	Model string
}

func (l *LMStudioBackend) ResolveReview(userInput string, filename string, tree []string) (TagResponse, error) {
	prompt := fmt.Sprintf(`A user is reviewing a file they want to move.
FILE: "%s"
USER DESCRIPTION of this file: "%s"

EXISTING FOLDERS:
%v

TASK:
Based on the user's description and the filename, determine which existing folder is the best match.
Suggest a NEW folder only if none of the existing ones are appropriate.
Return ONLY valid JSON in this format:
{
  "destination": "Folder/Name/",
  "is_new_folder": true/false,
  "confidence": 0.0-1.0,
  "reasoning": "brief explanation"
}`, filename, userInput, tree)

	chatReq := map[string]interface{}{
		"model": l.Model,
		"messages": []map[string]string{
			{"role": "system", "content": "You are a helpful assistant that outputs only JSON."},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.1,
	}

	return l.sendRequest(chatReq)
}

func (l *LMStudioBackend) TagContent(req TagRequest) (TagResponse, error) {
	// Build prompt
	prompt := fmt.Sprintf(`You are a file organiser. 
Given the filename: "%s"
Extension: "%s"
Content snippet: "%s"

EXISTING USER FOLDERS:
%v

PRIMARY CATEGORIES (Preferred):
%v

TASK:
Decide where this file should go and suggest 1-3 appropriate tags.
- You MUST prefer moving files into subfolders of the PRIMARY CATEGORIES.
- If an existing folder matches perfectly, use it.
- If you need a new subfolder, suggest it (e.g. "Documents/Personal/Tax").
- Return ONLY valid JSON in this format:
{
  "destination": "Relative/Path/To/Folder/",
  "tags": ["tag1", "tag2"],
  "is_new_folder": true/false,
  "confidence": 0.0-1.0,
  "reasoning": "brief explanation"
}`, req.Filename, req.Extension, req.ContentPeek, req.FolderTree, req.AllowedRoots)

	chatReq := map[string]interface{}{
		"model": l.Model,
		"messages": []map[string]string{
			{"role": "system", "content": "You are a helpful assistant that outputs only JSON."},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.1,
	}

	return l.sendRequest(chatReq)
}

func (l *LMStudioBackend) sendRequest(chatReq map[string]interface{}) (TagResponse, error) {
	reqBytes, _ := json.Marshal(chatReq)
	httpReq, _ := http.NewRequest("POST", l.Host+"/v1/chat/completions", bytes.NewBuffer(reqBytes))
	httpReq.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return TagResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return TagResponse{}, fmt.Errorf("LLM API returned status %d", resp.StatusCode)
	}

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return TagResponse{}, err
	}

	if len(chatResp.Choices) == 0 {
		return TagResponse{}, fmt.Errorf("no response from LLM")
	}

	var tr TagResponse
	rawContent := chatResp.Choices[0].Message.Content
	
	// Clean markdown block if present
	rawContent = strings.TrimPrefix(rawContent, "```json")
	rawContent = strings.TrimSuffix(rawContent, "```")
	rawContent = strings.TrimSpace(rawContent)

	if err := json.Unmarshal([]byte(rawContent), &tr); err != nil {
		return TagResponse{}, fmt.Errorf("failed to parse LLM JSON: %v. Raw content: %s", err, rawContent)
	}

	return tr, nil
}
