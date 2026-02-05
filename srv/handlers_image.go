package srv

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"srv.exe.dev/db/dbgen"
)

type ParsedImageTransaction struct {
	Date              string `json:"date"`
	Description       string `json:"description"`
	Amount            int64  `json:"amount"`
	CardLastFour      string `json:"card_last_four"`
	IsInstallment     bool   `json:"is_installment"`
	InstallmentMonths int64  `json:"installment_months"`
	IsDuplicate       bool   `json:"is_duplicate"`
}

type ParseImageResponse struct {
	Transactions []ParsedImageTransaction `json:"transactions"`
	Error        string                   `json:"error,omitempty"`
}

const imageParsePrompt = `이 이미지는 신용카드 이용내역 목록입니다. 각 거래를 JSON 배열로 추출해주세요.

각 거래에서 다음 정보를 추출해주세요:
- date: 거래일자 (YYYY-MM-DD 형식, 연도가 없으면 현재 연도 사용)
- description: 가맹점명/사용처
- amount: 금액 (숫자만, 원 단위)
- card_last_four: 카드 끝 4자리 (있으면)
- is_installment: 할부 여부 (일시불이면 false)
- installment_months: 할부 개월수 (일시불이면 0)

응답은 반드시 JSON 배열만 출력해주세요. 다른 설명은 넣지 마세요.
예시:
[{"date":"2026-01-31","description":"스타벅스","amount":5500,"card_last_four":"9869","is_installment":false,"installment_months":0}]`

// HandleAPIParseImage parses transaction image using Claude or GPT Vision API
func (s *Server) HandleAPIParseImage(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (max 10MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		s.writeJSON(w, ParseImageResponse{Error: "이미지 파일을 읽을 수 없습니다"})
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		s.writeJSON(w, ParseImageResponse{Error: "이미지 파일을 선택해주세요"})
		return
	}
	defer file.Close()

	// Read image data
	imageData, err := io.ReadAll(file)
	if err != nil {
		s.writeJSON(w, ParseImageResponse{Error: "이미지 읽기 실패"})
		return
	}

	// Determine media type
	mediaType := header.Header.Get("Content-Type")
	if mediaType == "" {
		mediaType = "image/png"
	}

	// Check for API keys - prefer OpenAI, fallback to Anthropic
	openaiKey := os.Getenv("OPENAI_API_KEY")
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")

	var transactions []ParsedImageTransaction

	if openaiKey != "" {
		transactions, err = parseImageWithGPT(openaiKey, imageData, mediaType)
	} else if anthropicKey != "" {
		transactions, err = parseImageWithClaude(anthropicKey, imageData, mediaType)
	} else {
		s.writeJSON(w, ParseImageResponse{Error: "이미지 분석 API 키가 설정되지 않았습니다. 환경변수 OPENAI_API_KEY 또는 ANTHROPIC_API_KEY를 설정해주세요."})
		return
	}

	if err != nil {
		s.writeJSON(w, ParseImageResponse{Error: fmt.Sprintf("이미지 분석 실패: %v", err)})
		return
	}

	// 중복 체크 및 카드 매칭
	cards, _ := dbgen.New(s.DB).GetAllCards(r.Context())
	for i := range transactions {
		tx := &transactions[i]
		var matchedCard *dbgen.Card
		if tx.CardLastFour != "" {
			for _, c := range cards {
				if strings.Contains(c.Name, tx.CardLastFour) {
					matchedCard = &c
					break
				}
			}
		}

		if matchedCard != nil {
			var exists bool
			err := s.DB.QueryRow(`
				SELECT EXISTS(
					SELECT 1 FROM transactions 
					WHERE card_id = ? 
					AND date(transaction_date) = date(?) 
					AND amount = ? 
					AND description = ?
					AND is_cancelled = 0
				)
			`, matchedCard.ID, tx.Date, tx.Amount, tx.Description).Scan(&exists)
			if err == nil && exists {
				tx.IsDuplicate = true
			}
		}
	}

	s.writeJSON(w, ParseImageResponse{Transactions: transactions})
}

func parseImageWithGPT(apiKey string, imageData []byte, mediaType string) ([]ParsedImageTransaction, error) {
	// Encode image to base64
	imageBase64 := base64.StdEncoding.EncodeToString(imageData)

	// Build OpenAI API request
	reqBody := map[string]interface{}{
		"model": "gpt-4o",
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": imageParsePrompt,
					},
					{
						"type": "image_url",
						"image_url": map[string]string{
							"url": fmt.Sprintf("data:%s;base64,%s", mediaType, imageBase64),
						},
					},
				},
			},
		},
		"max_tokens": 4096,
	}

	reqJSON, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(reqJSON))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API 호출 실패: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API 오류 (%d): %s", resp.StatusCode, string(body))
	}

	// Parse OpenAI response
	var gptResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&gptResp); err != nil {
		return nil, fmt.Errorf("응답 파싱 실패: %w", err)
	}

	if len(gptResp.Choices) == 0 {
		return nil, fmt.Errorf("빈 응답")
	}

	// Extract JSON from response text
	text := gptResp.Choices[0].Message.Content
	text = extractJSON(text)

	var transactions []ParsedImageTransaction
	if err := json.Unmarshal([]byte(text), &transactions); err != nil {
		return nil, fmt.Errorf("거래 데이터 파싱 실패: %w (응답: %s)", err, text)
	}

	// Normalize dates
	for i := range transactions {
		transactions[i].Date = normalizeDate(transactions[i].Date)
	}

	return transactions, nil
}

func parseImageWithClaude(apiKey string, imageData []byte, mediaType string) ([]ParsedImageTransaction, error) {
	// Encode image to base64
	imageBase64 := base64.StdEncoding.EncodeToString(imageData)

	// Build Claude API request
	reqBody := map[string]interface{}{
		"model":      "claude-sonnet-4-20250514",
		"max_tokens": 4096,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "image",
						"source": map[string]string{
							"type":       "base64",
							"media_type": mediaType,
							"data":       imageBase64,
						},
					},
					{
						"type": "text",
						"text": imageParsePrompt,
					},
				},
			},
		},
	}

	reqJSON, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(reqJSON))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API 호출 실패: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API 오류 (%d): %s", resp.StatusCode, string(body))
	}

	// Parse Claude response
	var claudeResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&claudeResp); err != nil {
		return nil, fmt.Errorf("응답 파싱 실패: %w", err)
	}

	if len(claudeResp.Content) == 0 {
		return nil, fmt.Errorf("빈 응답")
	}

	// Extract JSON from response text
	text := claudeResp.Content[0].Text
	text = extractJSON(text)

	var transactions []ParsedImageTransaction
	if err := json.Unmarshal([]byte(text), &transactions); err != nil {
		return nil, fmt.Errorf("거래 데이터 파싱 실패: %w (응답: %s)", err, text)
	}

	// Normalize dates
	for i := range transactions {
		transactions[i].Date = normalizeDate(transactions[i].Date)
	}

	return transactions, nil
}

// extractJSON tries to extract a JSON array from text that might have surrounding content
func extractJSON(text string) string {
	text = strings.TrimSpace(text)

	// Remove markdown code blocks if present
	if strings.HasPrefix(text, "```json") {
		text = strings.TrimPrefix(text, "```json")
		if idx := strings.LastIndex(text, "```"); idx != -1 {
			text = text[:idx]
		}
		text = strings.TrimSpace(text)
	} else if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```")
		if idx := strings.LastIndex(text, "```"); idx != -1 {
			text = text[:idx]
		}
		text = strings.TrimSpace(text)
	}

	// If it starts with [ and ends with ], assume it's already JSON
	if strings.HasPrefix(text, "[") && strings.HasSuffix(text, "]") {
		return text
	}

	// Try to find JSON array in the text
	re := regexp.MustCompile(`\[\s*\{[\s\S]*\}\s*\]`)
	if match := re.FindString(text); match != "" {
		return match
	}

	return text
}

// normalizeDate converts various date formats to YYYY-MM-DD
func normalizeDate(dateStr string) string {
	// Already in correct format
	if matched, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}$`, dateStr); matched {
		return dateStr
	}

	year := time.Now().Year()

	// Try MM/DD format
	if matched, _ := regexp.MatchString(`^\d{1,2}/\d{1,2}$`, dateStr); matched {
		parts := strings.Split(dateStr, "/")
		month, _ := strconv.Atoi(parts[0])
		day, _ := strconv.Atoi(parts[1])
		return fmt.Sprintf("%d-%02d-%02d", year, month, day)
	}

	// Try MM월 DD일 format
	re := regexp.MustCompile(`(\d{1,2})월\s*(\d{1,2})일`)
	if matches := re.FindStringSubmatch(dateStr); len(matches) > 2 {
		month, _ := strconv.Atoi(matches[1])
		day, _ := strconv.Atoi(matches[2])
		return fmt.Sprintf("%d-%02d-%02d", year, month, day)
	}

	// Return as-is if we can't parse
	return dateStr
}

// HandleAPICreateMultipleTransactions creates multiple transactions at once
func (s *Server) HandleAPICreateMultipleTransactions(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Transactions []struct {
			CardID             int64  `json:"card_id"`
			TransactionDate    string `json:"transaction_date"`
			Amount             int64  `json:"amount"`
			Description        string `json:"description"`
			CategoryID         *int64 `json:"category_id"`
			IsInstallment      int64  `json:"is_installment"`
			InstallmentMonths  *int64 `json:"installment_months"`
			InstallmentCurrent *int64 `json:"installment_current"`
			OriginalAmount     *int64 `json:"original_amount"`
		} `json:"transactions"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, 400, err.Error())
		return
	}

	var created []int64
	var skippedCount int
	for _, tx := range req.Transactions {
		// 중복 체크: 동일 카드, 날짜(시간 제외), 금액, 내용인 거래가 있는지 확인
		var exists bool
		err := s.DB.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM transactions 
				WHERE card_id = ? 
				AND date(transaction_date) = date(?) 
				AND amount = ? 
				AND description = ?
				AND is_cancelled = 0
			)
		`, tx.CardID, tx.TransactionDate, tx.Amount, tx.Description).Scan(&exists)

		if err == nil && exists {
			skippedCount++
			continue
		}

		result, err := s.DB.Exec(`
			INSERT INTO transactions (card_id, transaction_date, amount, description, category_id, is_installment, installment_months, installment_current, original_amount)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, tx.CardID, tx.TransactionDate, tx.Amount, tx.Description, tx.CategoryID, tx.IsInstallment, tx.InstallmentMonths, tx.InstallmentCurrent, tx.OriginalAmount)

		if err != nil {
			continue
		}
		id, _ := result.LastInsertId()
		created = append(created, id)
	}

	s.writeJSON(w, map[string]interface{}{
		"created_count": len(created),
		"created_ids":   created,
		"skipped_count": skippedCount,
	})
}
