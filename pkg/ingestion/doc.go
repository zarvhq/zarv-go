// Package ingestion sends events to the Zarv ingestion gateway, which lands
// each one in its bronze table.
//
// A Client is configured from the environment: ZARV_INGESTION_URL is the
// gateway's base URL and ZARV_INGESTION_KEY its Bearer key. New fails at once
// when either is missing or the URL is not an absolute http(s) URL.
//
//	client, err := ingestion.New(ingestion.Config{})
//	if err != nil {
//		log.Fatal(err) // names the variable that is missing
//	}
//
// Send posts one Event, the gateway's body as it is, and returns once the
// gateway accepts or refuses it. A refusal, a 202 that rejects the event
// included, is a *RefusedError and is not worth resending; a gateway that did
// not take the event within the attempts is an *UnavailableError, and the event
// can be sent again: the gateway identifies a row by its content.
package ingestion
