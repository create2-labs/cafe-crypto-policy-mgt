package metrics

import "testing"

func TestCatalogueCountersIncrement(t *testing.T) {
	AddCataloguePostureOrphans(0)
	AddCatalogueMalformedManifests(0)
	AddCataloguePostureOrphans(2)
	AddCatalogueMalformedManifests(1)
}
