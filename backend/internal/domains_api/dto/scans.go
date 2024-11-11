package dto

type ScanDomainRequest struct {
	DomainIds     []string `schema:"domainIds"`
	TemplateIds   []string `schema:"templateIds"`
	ScanAllNuclei bool     `schema:"scanAllNuclei"`
	Name          string   `schema:"name"`
}

type ScheduleScanRequest struct {
	DomainIds     []string `schema:"domainIds"`
	TemplateIds   []string `schema:"templateIds"`
	ScanAll       bool     `schema:"scanAll"`
	ScheduledDate string   `schema:"scheduledDate"`
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
	ScanTook    int64    `json:"scanTook"`
}

type GetAllMultiScansResponse struct {
	ID             string   `json:"id"`
	ScanIDs        []string `json:"scanIds"`
	Name           string   `json:"name"`
	TotalScans     int      `json:"totalScans"`
	CompletedScans int      `json:"completedScans"`
	FailedScans    int      `json:"failedScans"`
	Status         string   `json:"status"`
}

type GetScansByMultiScanIdResponse struct {
	MultiScanID    string                `json:"multiScanId"`
	Name           string                `json:"name"`
	Status         string                `json:"status"`
	TotalScans     int                   `json:"totalScans"`
	CompletedScans int                   `json:"completedScans"`
	FailedScans    int                   `json:"failedScans"`
	Scans          []GetAllScansResponse `json:"scans"`
}

type GetScanResponse struct {
	ID          string   `json:"id"`
	DomainId    string   `json:"domainId"`
	Domain      string   `json:"domain"`
	TemplateIds []string `json:"templateIds"`
	ScanDate    string   `json:"scanDate"`
	Status      string   `json:"status"`
	Error       string   `json:"error,omitempty"`
	S3ResultURL []string `json:"s3ResultUrl,omitempty"`
	ScanTook    int64    `json:"scanTook"`
}
