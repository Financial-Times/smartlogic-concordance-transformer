package smartlogicconcordance

type SmartlogicConcept struct {
	Concepts []Concept `json:"@graph"`
}

type Concept struct {
	Id string `json:"@id"`
	TmeIdentifiers []TmeId `json:"http://www.ft.com/ontology/TMEIdentifier"`
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