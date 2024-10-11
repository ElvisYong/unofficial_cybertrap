package dto

type DomainDeleteQuery struct {
	Id string `schema:"id"`
}

type DomainCreateQuery struct {
	Domain string `schema:"domain"`
	Page   int16  `schema:"page"`
}

type GetDomainResponse struct {
	ID         string `json:"id"`
	Domain     string `json:"domain"`
	UploadedAt string `json:"uploadedAt"`
	UserID     string `json:"userId"`
}
