package autodiscovery

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

type GroupBy string

const (
	GROUPBYALL        GroupBy = "all"
	GROUPBYINDIVIDUAL GroupBy = "individual"
	GROUPBYRESOURCE   GroupBy = "resource"
)

func (g GroupBy) Validate() error {
	valid := false
	for _, groupBy := range []GroupBy{
		GROUPBYALL,
		GROUPBYINDIVIDUAL,
		GROUPBYRESOURCE,
		"",
	} {
		if g == groupBy {
			valid = true
			break
		}
	}

	if !valid {
		err := fmt.Errorf("autodiscovery key, 'groupby' is wrongly set to %q, and must be one of [%q,%q,%q,%q]",
			g,
			"",
			GROUPBYALL,
			GROUPBYINDIVIDUAL,
			GROUPBYRESOURCE)

		logrus.Errorln(err)
		return err

	}
	return nil
}
