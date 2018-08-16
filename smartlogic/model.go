package smartlogic

type SmartlogicConcept struct {
	Concepts []Concept `json:"@graph"`
}

type Concept struct {
	Id                                string         `json:"@id"`
	Types                             []string       `json:"@type,omitempty"`
	TmeIdentifiersEditorial           []TmeId        `json:"http://www.ft.com/ontology/TMEIdentifier,omitempty"`
	TmeIdentifiersManagedLocation     []TmeId        `json:"http://www.ft.com/ontology/managedlocation/TMEIdentifier,omitempty"`
	FactsetIdentifiersEditorial       []FactsetId    `json:"http://www.ft.com/ontology/factsetIdentifier,omitempty"`
	FactsetIdentifiersManagedLocation []FactsetId    `json:"http://www.ft.com/ontology/managedlocation/factsetIdentifier,omitempty"`
	DbpediaIdentifiers                []LocationType `json:"http://www.ft.com/ontology/managedlocation/dbpediaId,omitempty"`
	GeonamesIdentifiers               []LocationType `json:"http://www.ft.com/ontology/managedlocation/geonamesId,omitempty"`
	WikidataIdentifiers               []LocationType `json:"http://www.ft.com/ontology/managedlocation/wikidataId,omitempty"`
}

func (c Concept) TmeIdentifiers() []TmeId {
	if c.TmeIdentifiersEditorial != nil {
		return c.TmeIdentifiersEditorial
	}
	return c.TmeIdentifiersManagedLocation
}

func (c Concept) FactsetIdentifiers() []FactsetId {
	if c.FactsetIdentifiersEditorial != nil {
		return c.FactsetIdentifiersEditorial
	}
	return c.FactsetIdentifiersManagedLocation
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
