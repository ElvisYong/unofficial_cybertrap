package dto

type ScanDomainRequest struct {
	DomainIds     []string `schema:"domainIds"`
	TemplateIds   []string `schema:"templateIds"`
	ScanAllNuclei bool     `schema:"scanAllNuclei"`
	Name          string   `schema:"name"`
}

type ScheduleSingleScanRequest struct {
	DomainId      string   `schema:"domainId"`
	TemplateIds   []string `schema:"templateIds"`
	ScheduledDate string   `schema:"scheduledDate"`
}

type DeleteScheduledScanRequest struct {
	ID string `schema:"ID"`
}

type ScheduleScanResponse struct {
	ID            string   `json:"id"`
	DomainId      string   `json:"domainId"`
	TemplateIds   []string `json:"templateIds"`
	ScheduledDate string   `json:"scheduledDate"`
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
