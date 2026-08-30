package dto

type LabFeature struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status" enum:"alpha,beta"`
	Available   bool   `json:"available"`
}
