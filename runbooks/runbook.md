# UPP - Smartlogic Concordance Transformer

This service listens for changes made in Smartlogic and persists the resulting concordance changes in the Cloud Neo4j using "concordances-rw-neo4j".

## Code

smartlogic-concordance-transform

## Primary URL

<https://upp-prod-publish-glb.upp.ft.com/__smartlogic-concordance-transformer/>

## Service Tier

Bronze

## Lifecycle Stage

Production

## Host Platform

AWS

## Architecture

This service will listen to the "SmartlogicConcept" Kafka topic for a notification of a change made in Smartlogic, verify whether the change concerns concordance, convert the JSON-LD in the message to a normalised UPP view of a concordance and finally send the JSON to the "concordances-rw-neo4j" service.

Checkout the application project repository for further details:
<https://github.com/Financial-Times/smartlogic-concordance-transformer>

## Contains Personal Data

No

## Contains Sensitive Data

No

## Failover Architecture Type

ActivePassive

## Failover Process Type

FullyAutomated

## Failback Process Type

Manual

## Failover Details

The service is deployed in the Publish clusters. The failover guide for the cluster is located here: <https://github.com/Financial-Times/upp-docs/tree/master/failover-guides/publishing-cluster>

## Data Recovery Process Type

NotApplicable

## Data Recovery Details

The service does not store data, so it does not require any data recovery steps.

## Release Process Type

PartiallyAutomated

## Rollback Process Type

Manual

## Release Details

The release is triggered by making a GitHub release which is then picked up by a Jenkins multibranch pipeline. The Jenkins pipeline should be manually started in order for it to deploy the helm package to the Kubernetes clusters.

## Key Management Process Type

NotApplicable

## Key Management Details

There is no key rotation procedure for this system.

## Monitoring

Look for the pods in the cluster health endpoint and click to see pod health and checks:

- EU cluster: <https://upp-prod-publish-eu.upp.ft.com/__health/__pods-health?service-name=smartlogic-concordance-transformer>
- US cluster: <https://upp-prod-publish-us.upp.ft.com/__health/__pods-health?service-name=smartlogic-concordance-transformer>

## First Line Troubleshooting

<https://github.com/Financial-Times/upp-docs/tree/master/guides/ops/first-line-troubleshooting>

## Second Line Troubleshooting

Please refer to the GitHub repository README for troubleshooting information.
