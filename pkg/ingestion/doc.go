// Package ingestion sends events to the Zarv ingestion gateway, which lands
// each one in its bronze table.
//
// With no configuration a Client sends to the gateway inside the cluster,
// DefaultURL, with no key. ZARV_INGESTION_URL points it at another gateway and
// ZARV_INGESTION_KEY adds a Bearer key for one that asks for it. New fails at
// once when the URL given is not an absolute http(s) URL.
//
//	client, err := ingestion.New(ingestion.Config{})
//	if err != nil {
//		log.Fatal(err) // names the variable with the bad value
//	}
//
// Send posts one Event, the gateway's body as it is, and returns once the
// gateway accepts or refuses it. A refusal, a 202 that rejects the event
// included, is a *RefusedError and is not worth resending; a gateway that did
// not take the event within the attempts is an *UnavailableError, and the event
// can be sent again: the gateway identifies a row by its content.
package ingestion
