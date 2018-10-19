package smartlogic

import (
	"encoding/json"
	"strings"
)

type SmartlogicConcept struct {
	Concepts []Concept `json:"@graph"`
}

type Concept struct {
	ID             string   `json:"@id"`
	Types          []string `json:"@type,omitempty"`
	currentConcept Concepter
}

type Concepter interface {
	TmeIdentifiers() []TmeId
	FactsetIdentifiers() []FactsetId
	DbpediaIdentifiers() []LocationType
	GeonamesIdentifiers() []LocationType
	WikidataIdentifiers() []LocationType
}

type ConceptML struct {
	TmeIdentifiersValue      []TmeId        `json:"http://www.ft.com/ontology/managedlocation/TMEIdentifier,omitempty"`
	FactsetIdentifiersValue  []FactsetId    `json:"http://www.ft.com/ontology/managedlocation/factsetIdentifier,omitempty"`
	DbpediaIdentifiersValue  []LocationType `json:"http://www.ft.com/ontology/managedlocation/dbpediaId,omitempty"`
	GeonamesIdentifiersValue []LocationType `json:"http://www.ft.com/ontology/managedlocation/geonamesId,omitempty"`
	WikidataIdentifiersValue []LocationType `json:"http://www.ft.com/ontology/managedlocation/wikidataId,omitempty"`
}

type ConceptEditorial struct {
	TmeIdentifiersValue      []TmeId        `json:"http://www.ft.com/ontology/TMEIdentifier,omitempty"`
	FactsetIdentifiersValue  []FactsetId    `json:"http://www.ft.com/ontology/factsetIdentifier,omitempty"`
	WikidataIdentifiersValue []LocationType `json:"http://www.ft.com/ontology/wikidataIdentifier,omitempty"`
	GeonamesIdentifiersValue []LocationType `json:"http://www.ft.com/ontology/geonamesIdentifier,omitempty"`
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

func (c *Concept) UnmarshalJSON(data []byte) error {
	aux := &struct {
		ID    string   `json:"@id"`
		Types []string `json:"@type,omitempty"`
		*ConceptML
		*ConceptEditorial
	}{}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if strings.Contains(aux.ID, "managedlocation") {
		if aux.ConceptML == nil {
			c.currentConcept = &ConceptML{}
		} else {
			c.currentConcept = aux.ConceptML
		}
	} else {
		if aux.ConceptEditorial == nil {
			c.currentConcept = &ConceptEditorial{}
		} else {
			c.currentConcept = aux.ConceptEditorial
		}
	}

	c.ID = aux.ID
	c.Types = aux.Types
	return nil
}

func (c Concept) TmeIdentifiers() []TmeId {
	return c.currentConcept.TmeIdentifiers()
}

func (c Concept) FactsetIdentifiers() []FactsetId {
	return c.currentConcept.FactsetIdentifiers()
}

func (c *Concept) DbpediaIdentifiers() []LocationType {
	return c.currentConcept.DbpediaIdentifiers()
}

func (c *Concept) GeonamesIdentifiers() []LocationType {
	return c.currentConcept.GeonamesIdentifiers()
}

func (c *Concept) WikidataIdentifiers() []LocationType {
	return c.currentConcept.WikidataIdentifiers()
}

func (c ConceptEditorial) DbpediaIdentifiers() []LocationType {
	return nil
}

func (c ConceptEditorial) GeonamesIdentifiers() []LocationType {
	return c.GeonamesIdentifiersValue
}

func (c ConceptEditorial) WikidataIdentifiers() []LocationType {
	return c.WikidataIdentifiersValue
}

func (c ConceptEditorial) TmeIdentifiers() []TmeId {
	return c.TmeIdentifiersValue
}

func (c ConceptEditorial) FactsetIdentifiers() []FactsetId {
	return c.FactsetIdentifiersValue
}

func (c ConceptML) TmeIdentifiers() []TmeId {
	return c.TmeIdentifiersValue
}

func (c ConceptML) FactsetIdentifiers() []FactsetId {
	return c.FactsetIdentifiersValue
}

func (c ConceptML) DbpediaIdentifiers() []LocationType {
	return c.DbpediaIdentifiersValue
}

func (c ConceptML) GeonamesIdentifiers() []LocationType {
	return c.GeonamesIdentifiersValue
}

func (c ConceptML) WikidataIdentifiers() []LocationType {
	return c.WikidataIdentifiersValue
}
