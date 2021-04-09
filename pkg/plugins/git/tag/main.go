package tag

import (
	"github.com/olblak/updateCli/pkg/plugins/version"
	"github.com/sirupsen/logrus"
)

// Tag contains git tag information
type Tag struct {
	Path          string         // Path contains the git repository path
	VersionFilter version.Filter // VersionFilter provides parameters to specify version pattern and its type like regex, semver, or just latest.
	Message       string         // Message associated to the git Tag
}

// Validate tests that tag struct is correctly configured
func (t *Tag) Validate() error {
	err := t.VersionFilter.Validate()
	if err != nil {
		return err
	}

	if len(t.Message) == 0 {
		logrus.Warningf("no git tag message specified")
		t.Message = "Generated by updatecli"
	}
	return nil
}