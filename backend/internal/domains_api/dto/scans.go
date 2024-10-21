package dto

type ScanDomainRequest struct {
	DomainIds     []string `schema:"domainIds"`
	TemplateIds   []string `schema:"templateIds"`
	ScanAllNuclei bool     `schema:"scanAllNuclei"`
	Name          string   `schema:"name"`
}

type ScheduleScanRequest struct {
	Id            string   `schema:"id"`
	DomainIds     []string `schema:"domainIds"`
	TemplateIds   []string `schema:"templateIds"`
	ScanAll       bool     `schema:"scanAll"`
	ScheduledDate string   `schema:"scheduledDate"`
}

type DeleteScheduledScanRequest struct {
	ID string `schema:"ID"`
}

type ScheduleScanResponse struct {
	ID            string                 `json:"id"`
	Domains       []GetDomainResponse    `json:"domains"`
	Templates     []GetTemplatesResponse `json:"templates"`
	ScanAll       bool                   `json:"scanAll"`
	ScheduledDate string                 `json:"scheduledDate"`
}

type GetAllScansResponse struct {
	ID          string   `json:"id"`
	DomainId    string   `json:"domainId"`
	Domain      string   `json:"domain"`
	TemplateIds []string `json:"templateIds"`
	ScanDate    string   `json:"scanDate"`
	Status      string   `json:"status"`
	Error       string   `json:"error,omitempty"`
	S3ResultURL []string `json:"s3ResultUrl,omitempty"`
}
