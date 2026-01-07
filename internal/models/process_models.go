package models

// ProcessRequest represents a simple processing request
type ProcessRequest struct {
	Arquivo string `json:"arquivo" validate:"required"` // URL do arquivo
}

// ProcessResponse represents the processing response
type ProcessResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	NovaURL   string `json:"nova_url,omitempty"`
	MediaType string `json:"media_type,omitempty"`
	FileID    string `json:"file_id,omitempty"`
}
