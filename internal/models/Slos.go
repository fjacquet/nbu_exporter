package models

// Slos is the response from GET /servicecatalog/slos (JSON:API).
//
// The 11.2 servicecatalog.yaml SLO schema (data[].attributes) exposes name,
// description, policyNamePrefix, workloadType, subscribeAllowed,
// subscriptionCount, schedules, capabilities and policyDefinition. There is NO
// per-SLO enforcement-type attribute: the "Enforcement Type" text in the spec
// refers to the endpoint's access-control model (Object-Level), not response
// data. The collector therefore emits a single unlabeled total count.
type Slos struct {
	Data []struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Attributes struct {
			Name string `json:"name"`
		} `json:"attributes"`
	} `json:"data"`
}
