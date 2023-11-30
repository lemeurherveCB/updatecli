package engine

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/updatecli/updatecli/pkg/core/cmdoptions"
	"github.com/updatecli/updatecli/pkg/core/config"
	"github.com/updatecli/updatecli/pkg/core/pipeline"
	"github.com/updatecli/updatecli/pkg/core/pipeline/action"
	"github.com/updatecli/updatecli/pkg/core/pipeline/autodiscovery"
	"github.com/updatecli/updatecli/pkg/core/pipeline/scm"
	"github.com/updatecli/updatecli/pkg/core/result"
	"github.com/updatecli/updatecli/pkg/core/version"
	"gopkg.in/yaml.v3"
)

// LoadAutoDiscovery tries to guess available pipelines based on specific directory
func (e *Engine) LoadAutoDiscovery(defaultEnabled bool) error {
	// Default Autodiscovery pipeline
	if defaultEnabled {
		logrus.Debugf("Default Autodiscovery crawlers enabled")
		var defaultPipeline pipeline.Pipeline

		err := defaultPipeline.Init(
			&config.Config{
				Spec: config.Spec{
					Name:          "Local AutoDiscovery",
					AutoDiscovery: autodiscovery.DefaultCrawlerSpecs,
				},
			},
			pipeline.Options{},
		)
		if err != nil {
			logrus.Errorln(err)
		} else {
			e.Pipelines = append(e.Pipelines, defaultPipeline)
		}
	}

	PrintTitle("Auto Discovery")

	for id, p := range e.Pipelines {
		if p.Config.Spec.AutoDiscovery.Crawlers == nil {
			continue
		}

		// TODO: To be removed once not experimental anymore
		if !cmdoptions.Experimental {
			logrus.Warningf("The 'autodiscovery' feature requires the flag experimental to work, such as:\n\t`updatecli manifest show --experimental`")
			return nil
		}

		PrintTitle(p.Name)

		var actionConfig *action.Config
		var autodiscoveryScm scm.Scm
		var autodiscoveryAction action.Action
		var found bool

		workDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed getting current working directory due to %v", err)
		}

		// Retrieve scm spec if it exists
		if len(p.Config.Spec.AutoDiscovery.ScmId) > 0 {
			autodiscoveryScm, found = p.SCMs[p.Config.Spec.AutoDiscovery.ScmId]

			if found {
				workDir = autodiscoveryScm.Handler.GetDirectory()
			}
		}

		/** Check for deprecated items **/
		if p.Config.Spec.AutoDiscovery.PullrequestId != "" {
			if p.Config.Spec.AutoDiscovery.ActionId != "" {
				return fmt.Errorf("the `autodiscovery.pullrequestid` and `autodiscovery.actionid` keywords are mutually exclusive. Please use only `autodiscovery.actionid` as `autodiscovery.pullrequestid` is deprecated")
			}

			logrus.Warningf("The `autodiscovery.pullrequestid` keyword is deprecated in favor of `autodiscovery.actionid`, please update this manifest. Updatecli will continue the execution while trying to translate `autodiscovery.pullrequestid` to `autodiscovery.actionid`.")

			p.Config.Spec.AutoDiscovery.ActionId = p.Config.Spec.AutoDiscovery.PullrequestId
			p.Config.Spec.AutoDiscovery.PullrequestId = ""
		}

		// Retrieve action spec if it exists
		if len(p.Config.Spec.AutoDiscovery.ActionId) > 0 {
			autodiscoveryAction, found = p.Actions[p.Config.Spec.AutoDiscovery.ActionId]

			if found {
				actionConfig = &autodiscoveryAction.Config
			}
		}

		c, err := autodiscovery.New(
			p.Config.Spec.AutoDiscovery, workDir)

		if err != nil {
			e.Pipelines[id].Report.Result = result.FAILURE
			logrus.Errorln(err)
			return err
		}

		errs := []error{}
		bytesManifests, err := c.Run()

		if err != nil {
			e.Pipelines[id].Report.Result = result.FAILURE
			logrus.Errorln(err)
			return err
		}

		if len(bytesManifests) == 0 {
			logrus.Infof("nothing detected")
		}

		for i := range bytesManifests {
			manifest := config.Spec{}

			// We expected manifest generated by the autodiscovery to use the yaml syntax
			err = yaml.Unmarshal(bytesManifests[i], &manifest)
			if err != nil {
				return err
			}

			switch p.Config.Spec.AutoDiscovery.GroupBy {
			/*
				By default if "group by" is not se then we fallback to all
				which means that we generate a single pipeline for all discovered manifests
				The goal is to have a "safe" default behavior and to avoid to accidentally generate
				dozens pullrequests for a single updatecli run
			*/
			case autodiscovery.GROUPBYALL, "":
				manifest.PipelineID = p.Config.Spec.PipelineID
			case autodiscovery.GROUPBYINDIVIDUAL:
				hash := sha256.New()
				/*
					We need to generate an uniq ID per individual pipeline
					but we shouldn't use the manifest of a pipeline
					because it may change over pipeline execution
					such as different source version filter

					Starting the id with the autodiscovery pipelineid looks enough
					to avoid collision
				*/
				_, err := io.WriteString(hash, p.Config.Spec.PipelineID+"/"+manifest.Name)
				if err != nil {
					logrus.Errorln(err)
				}
				manifest.PipelineID = fmt.Sprintf("%x", hash.Sum(nil))
			}

			manifest.SCMs = make(map[string]scm.Config)
			for scmId, sc := range p.SCMs {
				manifest.SCMs[scmId] = *sc.Config
			}

			if actionConfig != nil {
				manifest.Actions = make(map[string]action.Config)
				if (p.Config.Spec.AutoDiscovery.GroupBy == autodiscovery.GROUPBYALL ||
					p.Config.Spec.AutoDiscovery.GroupBy == "") &&
					actionConfig.Title == "" {
					/*
						Normally the config spec name should never be empty as it is a required field according the jsonschema spec
					*/
					actionConfig.Title = p.Config.Spec.Name
					if actionConfig.Title == "" {
						/*
							If Autodiscovery option "groupby" is set to "all" and if associated action title are set to "empty"
							then we want to be sure that the action title is not empty by using a generic title.
							Otherwise each pipeline generated by the autodiscovery will have a different title which will constantly update the pullrequest title.
						*/
						defaultActionTitle := "deps: bumping various version"
						logrus.Warningf("action title %q used by autodiscovery is empty, fallback to generic:\n\t=> %s",
							p.Config.Spec.AutoDiscovery.ActionId,
							defaultActionTitle)
						actionConfig.Title = defaultActionTitle
					}
				}
				manifest.Actions[p.Config.Spec.AutoDiscovery.ScmId] = *actionConfig
			}

			if manifest.Version != "" {
				manifest.Version = version.Version
			}

			newConfig := config.Config{
				Spec: manifest,
			}

			newPipeline := pipeline.Pipeline{}
			err = newPipeline.Init(&newConfig, e.Options.Pipeline)

			if err == nil {
				e.Pipelines = append(e.Pipelines, newPipeline)
				e.configurations = append(e.configurations, newConfig)
			} else {
				e.Pipelines[id].Report.Result = result.FAILURE
				// don't initially fail as init. of the pipeline still fails even with a successful validation
				err := fmt.Errorf("%q - %s", manifest.Name, err)
				errs = append(errs, err)
			}
			if len(errs) > 0 {
				e.Pipelines[id].Report.Result = result.FAILURE

				logrus.Errorf("Error(s) happened while generating Updatecli pipeline manifest")
				for i := range errs {
					logrus.Errorf("%v", errs[i])
				}
			}
		}

		e.Pipelines[id].Report.Result = result.SUCCESS

	}

	return nil

}