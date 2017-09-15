package smartlogic

type SmartlogicConcept struct {
	Concepts []Concept `json:"@graph"`
}

type Concept struct {
	Id             string   `json:"@id"`
	Types          []string `json:"@type,omitempty"`
	TmeIdentifiers []TmeId  `json:"http://www.ft.com/ontology/TMEIdentifier"`
}

type TmeId struct {
	Value string `json:"@value"`
}

type UppConcordance struct {
	ConceptUuid  string   `json:"uuid"`
	ConcordedIds []string `json:"concordedIds"`
}
