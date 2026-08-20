package domain

import "fmt"

// BusinessRuleError adalah pelanggaran aturan bisnis (BR-xx). Handler memetakannya
// ke HTTP 422 dengan field "rule" terisi (AGENTS.md bagian 4.3).
//
// Message TIDAK BOLEH memuat NIK, nomor dokumen, atau path foto (BR-11).
type BusinessRuleError struct {
	Rule    string
	Message string
}

func (e *BusinessRuleError) Error() string {
	return fmt.Sprintf("%s: %s", e.Rule, e.Message)
}

// NewBusinessRuleError membuat pelanggaran aturan bisnis. Pesannya wajib menyebut
// kode BR karena AC-04 memeriksa hal itu secara langsung.
func NewBusinessRuleError(rule, format string, args ...any) *BusinessRuleError {
	return &BusinessRuleError{Rule: rule, Message: fmt.Sprintf(format, args...)}
}

// ConfigError dipakai ketika parameter yang wajib ada di database belum diatur.
// Ini BUKAN alasan memakai nilai default: perhitungan berhenti (AGENTS.md Larangan 3).
type ConfigError struct {
	Message string
}

func (e *ConfigError) Error() string { return "konfigurasi: " + e.Message }

func NewConfigError(format string, args ...any) *ConfigError {
	return &ConfigError{Message: fmt.Sprintf(format, args...)}
}
