package srv

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type ParsedSMS struct {
	CardName    string `json:"card_name"`
	Amount      int64  `json:"amount"`
	Description string `json:"description"`
	Date        string `json:"date"`
	Time        string `json:"time"`
	IsInstallment     bool  `json:"is_installment"`
	InstallmentMonths int64 `json:"installment_months"`
	IsCancelled       bool  `json:"is_cancelled"`
}

func (s *Server) HandleParseSMS(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Text string `json:"text"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	
	result := parseSMS(req.Text)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func parseSMS(text string) ParsedSMS {
	result := ParsedSMS{}
	
	// Detect card company
	cardPatterns := map[string][]string{
		"KB국민카드": {"KB국민카드", "KB국민"},
		"우리카드":   {"우리(", "우리카드"},
		"삼성카드":   {"삼성(", "삼성카드"},
		"신한카드":   {"신한(", "신한카드"},
		"현대카드":   {"현대(", "현대카드"},
		"롯데카드":   {"롯데(", "롯데카드"},
		"하나카드":   {"하나(", "하나카드"},
		"NH농협카드": {"NH(", "NH농협"},
		"BC카드":    {"BC(", "BC카드"},
	}
	
	for cardName, patterns := range cardPatterns {
		for _, p := range patterns {
			if strings.Contains(text, p) {
				result.CardName = cardName
				break
			}
		}
		if result.CardName != "" {
			break
		}
	}
	
	// Extract amount (XX,XXX원 or X,XXX,XXX원)
	amountRe := regexp.MustCompile(`([\d,]+)원`)
	if matches := amountRe.FindStringSubmatch(text); len(matches) > 1 {
		amountStr := strings.ReplaceAll(matches[1], ",", "")
		result.Amount, _ = strconv.ParseInt(amountStr, 10, 64)
	}
	
	// Extract date (MM/DD)
	dateRe := regexp.MustCompile(`(\d{1,2})/(\d{1,2})`)
	if matches := dateRe.FindStringSubmatch(text); len(matches) > 2 {
		month, _ := strconv.Atoi(matches[1])
		day, _ := strconv.Atoi(matches[2])
		year := time.Now().Year()
		// If the month is greater than current month, it might be last year
		if month > int(time.Now().Month()) {
			year--
		}
		result.Date = time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local).Format("2006-01-02")
	}
	
	// Extract time (HH:MM)
	timeRe := regexp.MustCompile(`(\d{1,2}):(\d{2})`)
	if matches := timeRe.FindStringSubmatch(text); len(matches) > 2 {
		result.Time = matches[1] + ":" + matches[2]
	}
	
	// Check installment
	if strings.Contains(text, "일시불") {
		result.IsInstallment = false
	} else {
		installmentRe := regexp.MustCompile(`(\d+)개월`)
		if matches := installmentRe.FindStringSubmatch(text); len(matches) > 1 {
			result.IsInstallment = true
			result.InstallmentMonths, _ = strconv.ParseInt(matches[1], 10, 64)
		}
	}
	
	// Extract description (merchant name) - usually the last line or after date/time
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip lines with common patterns
		if line == "" || strings.Contains(line, "원") || strings.Contains(line, "승인") ||
			strings.Contains(line, "Web발신") || strings.Contains(line, "누적") ||
			strings.HasPrefix(line, "[") {
			continue
		}
		// Check if it looks like a merchant name
		if !strings.Contains(line, "/") && !strings.Contains(line, ":") {
			result.Description = line
		}
	}
	
	// For "우리카드" format, description is usually the merchant name line
	if result.CardName == "우리카드" {
		// Look for merchant after date/time line
		for i, line := range lines {
			if dateRe.MatchString(line) && i+1 < len(lines) {
				merchant := strings.TrimSpace(lines[i+1])
				if merchant != "" && !strings.Contains(merchant, "누적") {
					result.Description = merchant
					break
				}
			}
		}
	}
	
	// Check for cancellation
	if strings.Contains(text, "취소") {
		result.IsCancelled = true
	}
	
	return result
}
