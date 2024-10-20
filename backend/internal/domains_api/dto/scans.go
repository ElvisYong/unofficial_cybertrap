package dto

type ScanDomainRequest struct {
	DomainIDs   []string `schema:"domainIds"`
	TemplateIDs []string `schema:"templateIds"`
	ScanAll     bool     `schema:"scanAll"`
	Name        string   `schema:"name"`
}

type ScheduleSingleScanRequest struct {
	DomainID      string   `schema:"domainId"`
	TemplateIDs   []string `schema:"templateIds"`
	ScheduledDate string   `schema:"scheduledDate"`
}

type DeleteScheduledScanRequest struct {
	ID string `schema:"ID"`
}

type ScheduleScanResponse struct {
	ID            string   `json:"id"`
	DomainID      string   `json:"domainId"`
	TemplateIDs   []string `json:"templateIds"`
	ScheduledDate string   `json:"scheduledDate"`
}

type GetAllScansResponse struct {
	ID          string   `json:"id"`
	DomainID    string   `json:"domainId"`
	Domain      string   `json:"domain"`
	TemplateIDs []string `json:"templateIds"`
	ScanDate    string   `json:"scanDate"`
	Status      string   `json:"status"`
	Error       string   `json:"error,omitempty"`
	S3ResultURL []string `json:"s3ResultUrl,omitempty"`
}
