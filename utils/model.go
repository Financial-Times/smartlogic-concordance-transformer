package utils

type SmartlogicConcept struct {
	Concepts []Concept `json:"@graph"`
	Contexts  Context  `json:"@context"`
}

type Concept struct {
	Ids []Uuid `json:"sem:guid"`
	Identifiers []TmeId `json:"http://www.ft.com/ontology/TMEIdentifier"`
}

type Uuid struct {
	Value string `json:"@value"`
}

type TmeId struct {
	Language string `json:"@language"`
	Value string `json:"@value"`
}

type Context struct {}

type UppConcordance struct {
	ConceptUuid string `json:"uuid"`
	ConcordedIds []string `json:"concordedIds"`
}