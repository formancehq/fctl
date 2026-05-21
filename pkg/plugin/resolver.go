package plugin

// Resolution represents the outcome of resolving how to handle a service command.
type Resolution interface {
	resolution()
}

// UsePlugin indicates that an installed plugin should handle the command.
type UsePlugin struct {
	Plugin *LoadedPlugin
}

func (UsePlugin) resolution() {}

// UseBuiltIn indicates that the built-in command should handle it.
type UseBuiltIn struct{}

func (UseBuiltIn) resolution() {}

// NeedInstall indicates that a plugin is needed but not installed.
// The caller should trigger auto-discovery.
type NeedInstall struct {
	ServiceName    string
	ServiceVersion string
	PluginVersion  string
	RegistryPlugin *RegistryPlugin
}

func (NeedInstall) resolution() {}

// Resolve determines whether a service command should be handled by a plugin,
// the built-in implementation, or needs a plugin install.
//
// Resolution order:
//  1. If an installed plugin has a compatibleWith range matching the service version → UsePlugin
//  2. If no plugin matches but builtInCovers is true → UseBuiltIn
//  3. If neither, look up the registry for a compatible version → NeedInstall
//  4. If the registry has no match either → UseBuiltIn (fallback, let it fail naturally)
func Resolve(
	serviceName string,
	serviceVersion string,
	manager *PluginManager,
	registry *RegistryClient,
	builtInCovers bool,
) (Resolution, error) {
	// 1. Check installed plugins
	plugin := manager.FindPluginForService(serviceName, serviceVersion)
	if plugin != nil {
		return UsePlugin{Plugin: plugin}, nil
	}

	// 2. Built-in fallback
	if builtInCovers {
		return UseBuiltIn{}, nil
	}

	// 3. Check registry for auto-discovery
	if registry != nil {
		reg, err := registry.FetchRegistry()
		if err == nil {
			if regPlugin, ok := reg.Plugins[serviceName]; ok {
				bestVersion, _, err := regPlugin.FindBestVersion(serviceVersion)
				if err == nil {
					return NeedInstall{
						ServiceName:    serviceName,
						ServiceVersion: serviceVersion,
						PluginVersion:  bestVersion,
						RegistryPlugin: &regPlugin,
					}, nil
				}
			}
		}
	}

	// 4. Nothing works — fall back to built-in and let it fail naturally
	return UseBuiltIn{}, nil
}
