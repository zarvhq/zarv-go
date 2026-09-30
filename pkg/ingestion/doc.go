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
package ingestion
