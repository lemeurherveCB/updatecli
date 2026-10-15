package reports

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/Masterminds/semver/v3"
	"github.com/sirupsen/logrus"
	"github.com/updatecli/updatecli/pkg/plugins/utils/ci"
	"github.com/updatecli/updatecli/pkg/plugins/utils/gitgeneric"
	"github.com/updatecli/updatecli/pkg/plugins/utils/permalink"
)

// Action is a struct used to store the result of an action. It is used to generate pullrequest body
type Action struct {
	// ID is the unique identifier of the action
	ID string `xml:"id,attr" json:"id,omitempty"`
	// Title is the title of the action
	Title string `xml:"-" json:"title,omitempty"`
	// PipelineTitle is the title of the pipeline
	PipelineTitle string `xml:"h3,omitempty" json:"pipelineTitle,omitempty"`
	// Description is the description of the action
	Description string `xml:"p,omitempty" json:"description,omitempty"`
	// Targets is the list of targets IDs associated with the action
	Targets []ActionTarget `xml:"details,omitempty" json:"targets,omitempty"`
	// using a pointer to avoid empty tag
	PipelineURL *PipelineURL `xml:"a,omitempty" json:"pipelineURL,omitempty"`
	// ManifestURL is a permalink to the updatecli manifest responsible for this action, using a
	// pointer to avoid empty tag
	ManifestURL *ManifestURL `xml:"manifest,omitempty" json:"manifestURL,omitempty"`
	// Link is the URL of the action
	Link string `xml:"link,omitempty" json:"actionUrl,omitempty"`
}

// ActionTargetChangelog is a struct used to store a target changelog
type ActionTargetChangelog struct {
	// Title is the title of the changelog
	Title string `xml:"summary,omitempty" json:"title,omitempty"`
	// Description is the description of the changelog
	Description string `xml:"pre,omitempty" json:"description,omitempty"`
}

// PipelineURL is a struct used to store a pipeline URL
type PipelineURL struct {
	// URL is the URL of the pipeline
	URL string `xml:"href,attr"`
	// Name is the name of the pipeline
	Name string `xml:",chardata"`
}

// ManifestURL is a struct used to store a permalink to the manifest responsible for an action
type ManifestURL struct {
	// URL is the permalink to the manifest file
	URL string `xml:"href,attr"`
	// Name is the display name for the manifest URL, such as its path relative to its git repository
	Name string `xml:",chardata"`
}

// String show an action report formatted as a string
func (a *Action) String() string {
	a.sort()
	output, err := xml.MarshalIndent(a, "", "    ")
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}

	return string(output[:])
}

func (a *Action) Merge(sourceAction *Action, useDetailsFromSourceAction bool) {
	var c, d []ActionTarget

	useDetailsFromSourceActionTarget := useDetailsFromSourceAction
	switch len(a.Targets) > len(sourceAction.Targets) {
	case true:
		c = a.Targets
		d = sourceAction.Targets
	case false:
		d = a.Targets
		c = sourceAction.Targets
		useDetailsFromSourceActionTarget = !useDetailsFromSourceAction
	}

	for i := range d {
		targetFound := false
		for j := range c {
			if d[i].ID == c[j].ID {
				targetFound = true
				c[j].Merge(&d[i], useDetailsFromSourceActionTarget)
				break
			}
		}
		if !targetFound {
			c = append(c, d[i])
		}
	}

	if useDetailsFromSourceAction {
		a.PipelineTitle = sourceAction.PipelineTitle
		a.Description = sourceAction.Description
	}

	a.Targets = c
	a.sort()
}

func (a *Action) sort() {
	sort.Slice(
		a.Targets,
		func(i, j int) bool {
			return a.Targets[i].ID < a.Targets[j].ID
		})

	for id, target := range a.Targets {
		sort.SliceStable(
			target.Changelogs,
			func(i, j int) bool {
				titleI := target.Changelogs[i].Title
				titleJ := target.Changelogs[j].Title

				versionI, errI := semver.NewVersion(titleI)
				versionJ, errJ := semver.NewVersion(titleJ)

				// If both are valid versions, compare semantically (descending: newest first)
				if errI == nil && errJ == nil {
					return versionJ.LessThan(versionI)
				}

				// Fall back to string comparison for non-versions (ascending)
				return titleI < titleJ
			})
		a.Targets[id] = target
	}
}

// updateTargetDescriptions updates descriptions from being console friendly to markdown friendly
func (a *Action) updateTargetDescriptions() {
	for id, target := range a.Targets {
		d := target.Description
		d = strings.Replace(d, "\n\t*", "\n\n*", 1)
		d = strings.ReplaceAll(d, "\n\t*", "\n*")
		a.Targets[id].Description = d
	}
}

// ToActionsString show an action report formatted as a string
func (a Action) ToActionsString() string {
	a.sort()
	a.updateTargetDescriptions()

	output, err := xml.MarshalIndent(
		Actions{
			Actions: []Action{
				a,
			},
		}, "", "    ")
	if err != nil {
		logrus.Errorf("error: %v\n", err)
	}

	return string(output[:])
}

// ToActionsMarkdownString show an action report formatted as a string using markdown
func (a Action) ToActionsMarkdownString() string {
	a.updateTargetDescriptions()

	tmpl, err := template.New("actions").Parse(markdownReportTemplate)
	if err != nil {
		logrus.Errorf("error: %v\n", err)
	}

	manifest := bytes.Buffer{}
	if err := tmpl.Execute(&manifest, a); err != nil {
		logrus.Debugln(err)
		logrus.Errorf("error: %v\n", err)
	}
	return manifest.String()
}

// UpdatePipelineURL analyze the local environment to guess if Updatecli is executed from a CI pipeline
func (a *Action) UpdatePipelineURL() {
	detectedCi, err := ci.New()
	if err != nil {
		logrus.Debugf("No CI pipeline detected (%s)\n", err)
	}

	if detectedCi == nil {
		// No CI pipeline detected
		return
	}

	a.PipelineURL = &PipelineURL{
		Name: detectedCi.Name(),
		URL:  detectedCi.URL(),
	}
}

// UpdateManifestURL analyzes the git repository containing manifestFilename to build a permalink
// to it at its current commit, when the git remote points at a forge whose permalink format is known.
func (a *Action) UpdateManifestURL(manifestFilename string) {
	if manifestFilename == "" {
		return
	}

	absManifestFilename, err := filepath.Abs(manifestFilename)
	if err != nil {
		logrus.Debugf("resolving absolute path for manifest %q: %s", manifestFilename, err)
		return
	}
	// Resolve symlinks so the manifest path shares the same canonical form as the
	// repository root reported by go-git below (e.g. /var vs /private/var on macOS).
	if resolved, err := filepath.EvalSymlinks(absManifestFilename); err == nil {
		absManifestFilename = resolved
	}
	manifestDir := filepath.Dir(absManifestFilename)

	gitHandler := gitgeneric.GoGit{}

	// The manifest may live in a subdirectory of its git repository, so the repository
	// root, rather than the manifest's own directory, must be used to open the repository.
	repositoryRoot, err := gitHandler.RepositoryRoot(manifestDir)
	if err != nil {
		logrus.Debugf("no git repository root found for manifest %q: %s", manifestFilename, err)
		return
	}

	remotes, err := gitHandler.RemoteURLs(repositoryRoot)
	if err != nil || len(remotes) == 0 {
		logrus.Debugf("no git remote found for manifest %q: %s", manifestFilename, err)
		return
	}

	remoteURL, ok := remotes[gitgeneric.DefaultRemoteReferenceName]
	if !ok {
		for _, u := range remotes {
			remoteURL = u
			break
		}
	}

	commitHash, err := gitHandler.GetLatestCommitHash(repositoryRoot)
	if err != nil {
		logrus.Debugf("no git commit found for manifest %q: %s", manifestFilename, err)
		return
	}

	relativePath, err := filepath.Rel(repositoryRoot, absManifestFilename)
	if err != nil {
		logrus.Debugf("computing manifest %q path relative to repository root %q: %s", manifestFilename, repositoryRoot, err)
		return
	}
	relativePath = filepath.ToSlash(relativePath)

	manifestURL, ok := permalink.New(remoteURL, commitHash, relativePath)
	if !ok {
		logrus.Debugf("unknown git forge for remote %q, skipping manifest permalink", remoteURL)
		return
	}

	a.ManifestURL = &ManifestURL{
		Name: relativePath,
		URL:  manifestURL,
	}
}
