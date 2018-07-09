package smartlogic

type SmartlogicConcept struct {
	Concepts []Concept `json:"@graph"`
}

type Concept struct {
	Id                  string         `json:"@id"`
	Types               []string       `json:"@type,omitempty"`
	TmeIdentifiers      []TmeId        `json:"http://www.ft.com/ontology/TMEIdentifier,omitempty"`
	FactsetIdentifiers  []FactsetId    `json:"http://www.ft.com/ontology/factsetIdentifier,omitempty"`
	DbpediaIdentifiers  []LocationType `json:"http://www.ft.com/ontology/dbpediaId,omitempty"`
	GeonamesIdentifiers []LocationType `json:"http://www.ft.com/ontology/geonamesId,omitempty"`
	WikidataIdentifiers []LocationType `json:"http://www.ft.com/ontology/wikidataId,omitempty"`
}

type TmeId struct {
	Value string `json:"@value"`
}

type FactsetId struct {
	Language string `json:"@language"`
	Value    string `json:"@value"`
}

type UppConcordance struct {
	Authority    string        `json:"authority"`
	ConceptUuid  string        `json:"uuid"`
	ConcordedIds []ConcordedId `json:"concordances"`
}

type ConcordedId struct {
	Authority      string `json:"authority"`
	AuthorityValue string `json:"authorityValue,omitempty"`
	UUID           string `json:"uuid"`
}

type LocationType struct {
	Type  string `json:"@type"`
	Value string `json:"@value"`
}
