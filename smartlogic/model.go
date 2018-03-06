package smartlogic

type SmartlogicConcept struct {
	Concepts []Concept `json:"@graph"`
}

type Concept struct {
	Id                 string      `json:"@id"`
	Types              []string    `json:"@type,omitempty"`
	TmeIdentifiers     []TmeId     `json:"http://www.ft.com/ontology/TMEIdentifier"`
	FactsetIdentifiers []FactsetId `json:"http://www.ft.com/ontology/factsetIdentifier"`
}

type TmeId struct {
	Value string `json:"@value"`
}

type FactsetId struct {
	Language string `json:"@language"`
	Value    string `json:"@value"`
}

type UppConcordance struct {
	ConceptUuid  string        `json:"uuid"`
	ConcordedIds []ConcordedId `json:"concordances"`
}

type ConcordedId struct {
	Authority string `json:"authority"`
	UUID      string `json:"uuid"`
}
