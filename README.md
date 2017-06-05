# smartlogic-concordance-transformer

[![Circle CI](https://circleci.com/gh/Financial-Times/smartlogic-concordance-transformer/tree/master.png?style=shield)](https://circleci.com/gh/Financial-Times/smartlogic-concordance-transformer/tree/master) [![Go Report Card](https://goreportcard.com/badge/github.com/Financial-Times/smartlogic-concordance-transformer)](https://goreportcard.com/report/github.com/Financial-Times/smartlogic-concordance-transformer) [![Coverage Status](https://coveralls.io/repos/github/Financial-Times/smartlogic-concordance-transformer/badge.svg)](https://coveralls.io/github/Financial-Times/smartlogic-concordance-transformer)

## Introduction

This service will listen to Kafka for a notification of a change made in Smartlogic, verify whether the change concerns concordance, convert the JSON-LD in the message to a normalised UPP view of a concordance and finally send the JSON to the concordance-rw-dynamo.

## Installation

Download the source code, dependencies and test dependencies:

        go get -u github.com/kardianos/govendor
        go get -u github.com/Financial-Times/smartlogic-concordance-transformer
        cd $GOPATH/src/github.com/Financial-Times/smartlogic-concordance-transformer
        govendor sync
        go build .

## Running locally

1. Run the tests and install the binary:

        govendor sync
        govendor test -v -race
        go install

2. Run the binary (using the `help` flag to see the available optional arguments):

        $GOPATH/bin/smartlogic-concordance-transformer [--help]

Options:

        --app-system-code="smartlogic-concordance-transformer"  System Code of the application ($APP_SYSTEM_CODE)
        --app-name="smartlogic-concordance-transformer"         Application name ($APP_NAME)
        --port="8080"                                           Port to listen on ($APP_PORT)
        --kafka-address="http:localhost:9092"                   Kafka Address 
        --kafka-topic="smartlogic-concepts"                     Kafka Topic for Smartlogic concordances
        --kafka-group="smartlogic-concordance-transformer"      Kafka Group for this service 
        --writer-endpoint="http://localhost:8080/"              Routing for the RW App                             
        
        
## Build and deployment

* Built by Docker Hub on merge to master: [coco/smartlogic-concordance-transformer](https://hub.docker.com/r/coco/smartlogic-concordance-transformer/)
* CI provided by CircleCI: [smartlogic-concordance-transformer](https://circleci.com/gh/Financial-Times/smartlogic-concordance-transformer)

## Service endpoints
See the api/api.yml for the swagger definitions of the endpoints

### POST /transform
This endpoint is for testing and help ongoing support. This endpoint only transforms the JSON-LD payload and returns the UPP source representation but doesn’t send it on down the pipeline to the concordance-rw-s3

Using curl:

    curl -X POST -i https://user:pass@pub-prod-up.ft.com/__smartlogic-concordance-transformer/transform/send --d @payload.txt --header "Content-Type:application/json"

Payload.txt:

    {
        "@graph": [
            {
                "@id": "http://www.ft.com/thing/2d3e16e0-61cb-4322-8aff-3b01c59f4daa",
                "@type": [
                    "http://www.ft.com/ontology/product/Brand"
                ],
                "http://www.ft.com/ontology/TMEIdentifier": [
                    {
                        "@value": "YzhlNzZkYTctMDJiNy00NTViLTk3NmYtNmJjYTE5NDEyM2Yw-QnJhbmRz"
                    }
                ],
                "http://www.ft.com/ontology/_logoURL": [
                    {
                        "@value": "http://im.ft-static.com/content/images/d5ffade2-99ea-11e6-8f9b-70e3cabccfae.png"
                    }
                ],
                "http://www.ft.com/ontology/description": [
                    {
                        "@language": "en",
                        "@value": "<p>Lex is a premium daily commentary service from the Financial Times. It is the oldest and arguably the most influential business and finance column of its kind in the world. It helps readers make better investment decisions by highlighting key emerging risks and opportunities.</p>"
                    }
                ],
                "http://www.ft.com/ontology/hasSubBrand": [
                    {
                        "@id": "http://www.ft.com/thing/e363dfb8-f6d9-4f2c-beba-5162b334272b"
                    }
                ],
                "http://www.ft.com/ontology/strapline": [
                    {
                        "@language": "en",
                        "@value": "FT's agenda-setting column on business and finance"
                    }
                ],
                "http://www.ft.com/ontology/subBrandOf": [
                    {
                        "@id": "http://www.ft.com/thing/dbb0bdae-1f0c-11e4-b0cb-b2227cce2b54"
                    }
                ],
                "sem:guid": [
                    {
                        "@value": "2d3e16e0-61cb-4322-8aff-3b01c59f4daa"
                    }
                ],
                "skosxl:prefLabel": [
                    {
                        "@id": "http://www.ft.com/thing/2d3e16e0-61cb-4322-8aff-3b01c59f4daa/Lex_en",
                        "skosxl:literalForm": [
                            {
                                "@language": "en",
                                "@value": "Lex"
                            }
                        ]
                    }
                ]
            }
        ],
        "@context": {
            ...
          
        }
    }


The expected response will give us a UPP source system representation of this smart logic concordance

e.g
    
    HTTP/1.1 200 OK
    Content-Type: application/json
    X-Request-Id: transaction ID, e.g. tid_etmIWTJVeA
    {
      "uuid": "2d3e16e0-61cb-4322-8aff-3b01c59f4daa",
      "concordedIds": [
    	"70f4732b-7f7d-30a1-9c29-0cceec23760e"
      ]
    }

Based on the following [google doc](https://docs.google.com/document/d/1-8Yv1ob6qjAOzfU1ngEOeXJDGq_zP7pLM7F5HnORCoM/edit#).

## Utility endpoints
### POST /transform/send
Transforms smartlogic payload into the upp representation of concordance and sends result to concordances-rw-s3

Using curl:

    curl -X POST -i https://user:pass@pub-prod-up.ft.com/__smartlogic-concordance-transformer/transform/send --d @payload.txt --header "Content-Type:application/json"

Payload.txt:


    {
        "@graph": [
            {
                "@id": "http://www.ft.com/thing/2d3e16e0-61cb-4322-8aff-3b01c59f4daa",
                "@type": [
                    "http://www.ft.com/ontology/product/Brand"
                ],
                "http://www.ft.com/ontology/TMEIdentifier": [
                    {
                        "@value": "YzhlNzZkYTctMDJiNy00NTViLTk3NmYtNmJjYTE5NDEyM2Yw-QnJhbmRz"
                    }
                ],
                "http://www.ft.com/ontology/_logoURL": [
                    {
                        "@value": "http://im.ft-static.com/content/images/d5ffade2-99ea-11e6-8f9b-70e3cabccfae.png"
                    }
                ],
                "http://www.ft.com/ontology/description": [
                    {
                        "@language": "en",
                        "@value": "<p>Lex is a premium daily commentary service from the Financial Times. It is the oldest and arguably the most influential business and finance column of its kind in the world. It helps readers make better investment decisions by highlighting key emerging risks and opportunities.</p>"
                    }
                ],
                "http://www.ft.com/ontology/hasSubBrand": [
                    {
                        "@id": "http://www.ft.com/thing/e363dfb8-f6d9-4f2c-beba-5162b334272b"
                    }
                ],
                "http://www.ft.com/ontology/strapline": [
                    {
                        "@language": "en",
                        "@value": "FT's agenda-setting column on business and finance"
                    }
                ],
                "http://www.ft.com/ontology/subBrandOf": [
                    {
                        "@id": "http://www.ft.com/thing/dbb0bdae-1f0c-11e4-b0cb-b2227cce2b54"
                    }
                ],
                "sem:guid": [
                    {
                        "@value": "2d3e16e0-61cb-4322-8aff-3b01c59f4daa"
                    }
                ],
                "skosxl:prefLabel": [
                    {
                        "@id": "http://www.ft.com/thing/2d3e16e0-61cb-4322-8aff-3b01c59f4daa/Lex_en",
                        "skosxl:literalForm": [
                            {
                                "@language": "en",
                                "@value": "Lex"
                            }
                        ]
                    }
                ]
            }
        ],
        "@context": {
            ...
          
        }
    }


Based on the following [google doc](https://docs.google.com/document/d/1-8Yv1ob6qjAOzfU1ngEOeXJDGq_zP7pLM7F5HnORCoM/edit#).

## Healthchecks
Admin endpoints are:

`/__gtg`

`/__health`

`/__build-info`

`/__ping`

There are several checks performed:

* Checks that a connection can be made to the concordances-rw-s3 service

### Logging

* The application uses [logrus](https://github.com/Sirupsen/logrus); the log file is initialised in [main.go](main.go).
* Logging requires an `env` app parameter, for all environments other than `local` logs are written to file.
* When running locally, logs are written to console. If you want to log locally to file, you need to pass in an env parameter that is != `local`.
* NOTE: `/__build-info` and `/__gtg` endpoints are not logged as they are called every second from varnish/vulcand and this information is not needed in logs/splunk.
