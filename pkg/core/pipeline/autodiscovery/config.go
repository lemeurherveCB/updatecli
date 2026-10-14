package autodiscovery

// Config defines autodiscovery parameters
type Config struct {
	/*
		Crawlers defines a map of crawler configuration where the key represent the crawler type
	*/
	Crawlers CrawlersConfig `yaml:",omitempty"`
	/*
		scmid is a unique identifier used to retrieve the scm configuration from the configuration file.
	*/
	ScmId string `yaml:",omitempty"`
	/*
		actionid is a unique identifier used to retrieve the action configuration from the configuration file.
	*/
	ActionId string `yaml:",omitempty"`
	/*
		groupby specifies how to group pipeline. The Accepted is one of "all", "individual", "resource". Default is "all"

		remark:
			* "all" generates a single pipeline for all discovered manifests, opening a single pull request.
			* "individual" generates one pipeline per discovered manifest, opening one pull request per manifest.
			* "resource" generates one pipeline per discovered resource, where a resource is the dependency a
			  manifest tracks (e.g. a Helm chart, a Maven artifact, an npm package, a container image). Every
			  manifest tracking the same resource, however many times or files it was discovered in, is routed
			  to the same pipeline and opens a single pull request for that resource. A manifest discovered
			  with more than one source falls back to the "individual" behavior, since it has no single
			  resource identity to group by.

		default:
			all
	*/
	GroupBy GroupBy
	// !Deprecated in favor of `actionid`
	PullrequestId string `yaml:",omitempty"`
}

// CrawlersConfig is a custom type used to generated the jsonschema.
type CrawlersConfig map[string]any
