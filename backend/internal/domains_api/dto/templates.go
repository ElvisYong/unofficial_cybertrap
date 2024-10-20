package dto

type TemplateDeleteQuery struct {
	Id string `schema:"id"`
}

type GetTemplatesResponse struct {
	ID          string                 `json:"id"`
	TemplateID  string                 `json:"templateId"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	S3URL       string                 `json:"s3Url"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Type        string                 `json:"type"`
	CreatedAt   string                 `json:"createdAt"`
}
